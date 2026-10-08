package handlers

import (
	"errors"
	"math"
	"net/http"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type rentalPositionInputError struct{ message string }

func (e *rentalPositionInputError) Error() string { return e.message }

type rentalPositionConflictError struct{ message string }

func (e *rentalPositionConflictError) Error() string { return e.message }
func rentalPositionErrorStatus(err error) int {
	var input *rentalPositionInputError
	var conflict *rentalPositionConflictError
	if errors.As(err, &input) {
		return http.StatusBadRequest
	}
	if errors.As(err, &conflict) {
		return http.StatusConflict
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

type rentalPositionPricing struct {
	ID            uint
	Name          string
	RentalPrice   *float64
	CustomerPrice *float64
}

func validRentalPrice(price *float64) bool {
	return price != nil && !math.IsNaN(*price) && !math.IsInf(*price, 0) && *price >= 0 && *price <= 9999999999.99 && math.Abs(*price*100-math.Round(*price*100)) < 0.00001
}

func loadRentalPositionPricing(tx *gorm.DB, equipmentID uint) (rentalPositionPricing, error) {
	var pricing rentalPositionPricing
	result := tx.Raw("SELECT id,name,rental_price,customer_price FROM rental_equipment WHERE id=? AND is_active=true FOR SHARE", equipmentID).Scan(&pricing)
	if result.Error != nil {
		return pricing, result.Error
	}
	if result.RowsAffected != 1 {
		return pricing, &rentalPositionInputError{"active rental equipment required"}
	}
	if !validRentalPrice(pricing.RentalPrice) {
		return pricing, &rentalPositionInputError{"catalog_rental_price required"}
	}
	if !validRentalPrice(pricing.CustomerPrice) {
		return pricing, &rentalPositionInputError{"catalog_customer_price required"}
	}
	return pricing, nil
}

// The cost ledger retains supplier prices and duration; only the position earns revenue.
// Call on the owning transaction, after locking the job before positions and costs.
func createRentalPosition(tx *gorm.DB, pos *models.JobPosition, days int64, notes string, repair bool) error {
	if pos.RentalEquipmentID == nil || *pos.RentalEquipmentID == 0 {
		return &rentalPositionInputError{"rental equipment required"}
	}
	pricing, err := loadRentalPositionPricing(tx, *pos.RentalEquipmentID)
	if err != nil {
		return err
	}
	var existing int64
	if err := tx.Model(&models.JobPosition{}).Where("job_id = ? AND position_type = 'rental' AND rental_equipment_id = ?", pos.JobID, *pos.RentalEquipmentID).Count(&existing).Error; err != nil {
		return err
	}
	if existing != 0 {
		return &rentalPositionConflictError{"rental position already exists; reconcile it explicitly"}
	}
	var cost struct {
		PositionID *uint
		Quantity   float64
		DaysUsed   int64
	}
	result := tx.Table("job_rental_equipment").Where("job_id = ? AND equipment_id = ?", pos.JobID, *pos.RentalEquipmentID).Scan(&cost)
	if result.Error != nil {
		return result.Error
	}
	if repair {
		if result.RowsAffected != 1 || cost.PositionID != nil || cost.Quantity != pos.Quantity || cost.DaysUsed != days {
			return &rentalPositionConflictError{"exact unlinked legacy assignment required"}
		}
	} else if result.RowsAffected != 0 {
		return &rentalPositionConflictError{"existing supplier assignment requires explicit repair"}
	}
	if days < 1 || days > 365 || math.Trunc(pos.Quantity) != pos.Quantity || pos.Quantity < 1 || pos.Quantity > 1000 {
		return &rentalPositionInputError{"whole quantity 1–1000 and rental days 1–365 required"}
	}
	if !repair {
		var multiply bool
		if err := tx.Raw("SELECT multiply_by_days FROM jobs WHERE jobid=?", pos.JobID).Scan(&multiply).Error; err != nil {
			return err
		}
		factor := int64(1)
		if multiply {
			factor = days
		}
		if *pricing.RentalPrice*pos.Quantity*float64(factor) > 9999999999.99 {
			return &rentalPositionInputError{"bounded_total_cost required"}
		}
	}
	pos.Description = pricing.Name
	pos.UnitPrice = *pricing.CustomerPrice
	pos.FollowDayFactor = 0
	// Explicit values preserve zero tax/follow-day factors instead of GORM defaults.
	if err := tx.Raw(`INSERT INTO job_positions(job_id,position_type,rental_equipment_id,description,quantity,unit,unit_price,follow_day_factor,discount_percent,discount_amount,tax_rate,sort_order)
	 VALUES(?,'rental',?,?,?,?,?,0,?,?,?,?) RETURNING position_id`, pos.JobID, *pos.RentalEquipmentID, pos.Description, pos.Quantity, pos.Unit, pos.UnitPrice, pos.DiscountPercent, pos.DiscountAmount, pos.TaxRate, pos.SortOrder).Scan(&pos.PositionID).Error; err != nil {
		return err
	}
	if repair {
		// Infer the captured unit cost from the stored amount, never from today's catalog.
		return tx.Exec(`UPDATE job_rental_equipment c SET position_id=?,rental_unit_price=COALESCE(c.rental_unit_price,c.total_cost/c.quantity::numeric/CASE WHEN j.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END)
		 FROM jobs j WHERE j.jobid=c.job_id AND c.job_id=? AND c.equipment_id=?`, pos.PositionID, pos.JobID, *pos.RentalEquipmentID).Error
	}
	return tx.Exec(`INSERT INTO job_rental_equipment(job_id,equipment_id,position_id,quantity,days_used,rental_unit_price,total_cost,notes)
	 SELECT j.jobid,?, ?, ?, ?, re.rental_price,round(re.rental_price*?::numeric*CASE WHEN j.multiply_by_days THEN ? ELSE 1 END,2),?
	 FROM jobs j JOIN rental_equipment re ON re.id=? WHERE j.jobid=?`, *pos.RentalEquipmentID, pos.PositionID, pos.Quantity, days, pos.Quantity, days, notes, *pos.RentalEquipmentID, pos.JobID).Error
}

func lockRentalPositionJob(tx *gorm.DB, jobID uint) error {
	var id uint
	result := tx.Raw("SELECT jobid FROM jobs WHERE jobid=? AND deleted_at IS NULL FOR UPDATE", jobID).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
