package handlers

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go-barcode-webapp/internal/logger"
)

type PositionHandler struct {
	positionRepo    *repository.PositionRepository
	jobRepo         *repository.JobRepository
	requirementRepo *repository.RequirementRepository
	db              *gorm.DB
}

func NewPositionHandler(positionRepo *repository.PositionRepository, jobRepo *repository.JobRepository, requirementRepo *repository.RequirementRepository, db *gorm.DB) *PositionHandler {
	if err := ensureJobPriceColumns(db); err != nil {
		logger.LogInfo("warning: failed to ensure job price columns: %v", err)
	}
	return &PositionHandler{
		positionRepo:    positionRepo,
		jobRepo:         jobRepo,
		requirementRepo: requirementRepo,
		db:              db,
	}
}

func ensureJobPriceColumns(db *gorm.DB) error {
	db.Exec(`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS multiply_by_days BOOLEAN NOT NULL DEFAULT TRUE`)
	db.Exec(`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS prices_include_tax BOOLEAN NOT NULL DEFAULT FALSE`)
	return nil
}

func (h *PositionHandler) syncRequirements(jobID uint) {
	var positions []models.JobPosition
	if err := h.db.Where("job_id = ? AND position_type = 'product'", jobID).Find(&positions).Error; err != nil {
		logger.LogInfo("syncRequirements: query failed for job %d: %v", jobID, err)
		return
	}
	reqs := make([]models.JobProductRequirement, 0, len(positions))
	for _, pos := range positions {
		if pos.ProductID == nil {
			continue
		}
		qty := int(math.Round(pos.Quantity))
		if qty < 1 {
			qty = 1
		}
		reqs = append(reqs, models.JobProductRequirement{
			JobID:     jobID,
			ProductID: *pos.ProductID,
			Quantity:  qty,
		})
	}
	if err := h.requirementRepo.SaveRequirements(jobID, reqs); err != nil {
		logger.LogInfo("syncRequirements: save failed for job %d: %v", jobID, err)
	}
}

func (h *PositionHandler) GetPositions(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

type CreatePositionInput struct {
	PositionType      string   `json:"position_type" binding:"required,oneof=product service rental package"`
	ProductID         *uint    `json:"product_id"`
	ServiceItemID     *uint    `json:"service_item_id"`
	RentalEquipmentID *uint    `json:"rental_equipment_id"`
	Description       string   `json:"description"`
	Quantity          float64  `json:"quantity"`
	Unit              string   `json:"unit"`
	UnitPrice         float64  `json:"unit_price"`
	FollowDayFactor   *float64 `json:"follow_day_factor"`
	DiscountPercent   float64  `json:"discount_percent"`
	DiscountAmount    float64  `json:"discount_amount"`
	TaxRate           *float64 `json:"tax_rate"`
}

func (h *PositionHandler) CreatePosition(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var input CreatePositionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	if input.Unit == "" {
		input.Unit = "Stück"
	}

	followDayFactor := 0.5
	if input.FollowDayFactor != nil {
		followDayFactor = *input.FollowDayFactor
	}
	if input.PositionType == "service" {
		followDayFactor = 0
	}
	// rental and package have no follow_day_factor
	if input.PositionType == "rental" || input.PositionType == "package" {
		followDayFactor = 0
	}

	taxRate := 19.0
	if input.TaxRate != nil {
		taxRate = *input.TaxRate
	}

	nextOrder, _ := h.positionRepo.GetNextSortOrder(uint(jobID))

	pos := models.JobPosition{
		JobID:             uint(jobID),
		PositionType:      input.PositionType,
		ProductID:         input.ProductID,
		ServiceItemID:     input.ServiceItemID,
		RentalEquipmentID: input.RentalEquipmentID,
		Description:       input.Description,
		Quantity:          input.Quantity,
		Unit:              input.Unit,
		UnitPrice:         input.UnitPrice,
		FollowDayFactor:   followDayFactor,
		DiscountPercent:   input.DiscountPercent,
		DiscountAmount:    input.DiscountAmount,
		TaxRate:           taxRate,
		SortOrder:         nextOrder,
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&pos).Error; err != nil {
			return err
		}
		return syncJobRevenue(tx, uint(jobID))
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	created, _ := h.positionRepo.GetByID(pos.PositionID)

	if input.PositionType == "product" {
		h.syncRequirements(uint(jobID))
	}

	c.JSON(http.StatusCreated, gin.H{"position": created})
}

type UpdatePositionInput struct {
	Description     *string  `json:"description"`
	Quantity        *float64 `json:"quantity"`
	Unit            *string  `json:"unit"`
	UnitPrice       *float64 `json:"unit_price"`
	FollowDayFactor *float64 `json:"follow_day_factor"`
	DiscountPercent *float64 `json:"discount_percent"`
	DiscountAmount  *float64 `json:"discount_amount"`
	TaxRate         *float64 `json:"tax_rate"`
}

func (h *PositionHandler) UpdatePosition(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}
	jobID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	pos, err := h.positionRepo.GetByID(uint(posID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	if pos.JobID != uint(jobID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	var input UpdatePositionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Description != nil {
		pos.Description = *input.Description
	}
	if input.Quantity != nil {
		pos.Quantity = *input.Quantity
	}
	if input.Unit != nil {
		pos.Unit = *input.Unit
	}
	if input.UnitPrice != nil {
		pos.UnitPrice = *input.UnitPrice
	}
	if input.FollowDayFactor != nil {
		pos.FollowDayFactor = *input.FollowDayFactor
	}
	if input.DiscountPercent != nil {
		pos.DiscountPercent = *input.DiscountPercent
	}
	if input.DiscountAmount != nil {
		pos.DiscountAmount = *input.DiscountAmount
	}
	if input.TaxRate != nil {
		pos.TaxRate = *input.TaxRate
	}
	pos.UpdatedAt = time.Now()

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(pos).Error; err != nil {
			return err
		}
		return syncJobRevenue(tx, pos.JobID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := h.positionRepo.GetByID(pos.PositionID)

	if pos.PositionType == "product" {
		h.syncRequirements(uint(jobID))
	}

	c.JSON(http.StatusOK, gin.H{"position": updated})
}

func (h *PositionHandler) DeletePosition(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}
	pos, err := h.positionRepo.GetByID(uint(posID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	if pos.JobID != uint(jobID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("position_id = ?", uint(posID)).Delete(&models.JobPosition{}).Error; err != nil {
			return err
		}
		return syncJobRevenue(tx, pos.JobID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if pos.PositionType == "product" {
		h.syncRequirements(pos.JobID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "position deleted"})
}

type ReorderInput struct {
	PositionIDs []uint `json:"position_ids" binding:"required"`
}

func (h *PositionHandler) ReorderPositions(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var input ReorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.positionRepo.Reorder(uint(jobID), input.PositionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reordered"})
}

type AssignDeviceInput struct {
	DeviceID  string `json:"device_id" binding:"required"`
	ScannedBy string `json:"scanned_by"`
}

func (h *PositionHandler) AssignDevice(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	var input AssignDeviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.positionRepo.AssignDevice(uint(posID), input.DeviceID, input.ScannedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "device assigned"})
}

func (h *PositionHandler) RemoveDevice(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}
	deviceID := c.Param("devId")

	if err := h.positionRepo.RemoveDevice(uint(posID), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device removed"})
}

func (h *PositionHandler) GetPicklist(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetPicklist(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type PicklistItem struct {
		PositionID  uint     `json:"position_id"`
		ProductID   *uint    `json:"product_id"`
		ProductName string   `json:"product_name"`
		Needed      int      `json:"needed"`
		Scanned     int      `json:"scanned"`
		Remaining   int      `json:"remaining"`
		DeviceIDs   []string `json:"device_ids"`
	}

	items := make([]PicklistItem, 0, len(positions))
	for _, p := range positions {
		productName := ""
		if p.Product != nil {
			productName = p.Product.Name
		}
		needed := int(p.Quantity)
		scanned := len(p.Devices)
		deviceIDs := make([]string, 0, len(p.Devices))
		for _, d := range p.Devices {
			deviceIDs = append(deviceIDs, d.DeviceID)
		}
		items = append(items, PicklistItem{
			PositionID:  p.PositionID,
			ProductID:   p.ProductID,
			ProductName: productName,
			Needed:      needed,
			Scanned:     scanned,
			Remaining:   max(0, needed-scanned),
			DeviceIDs:   deviceIDs,
		})
	}

	c.JSON(http.StatusOK, gin.H{"picklist": items})
}

func (h *PositionHandler) GetTotals(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	eventDays := calcEventDays(job.StartDate, job.EndDate)

	subtotalInvoice := 0.0
	totals := revenueAmounts{}
	for _, p := range positions {
		invoiceAmount := positionInvoiceRevenue(p, job.MultiplyByDays, eventDays)
		amounts := splitInvoiceRevenue(invoiceAmount, p.TaxRate, job.PricesIncludeTax)
		subtotalInvoice += invoiceAmount
		totals.Net += amounts.Net
		totals.Gross += amounts.Gross
	}

	discountFactor := jobDiscountFactor(subtotalInvoice, job.Discount, job.DiscountType)
	subtotalNet := totals.Net
	totals = totals.scale(discountFactor)
	globalDiscount := subtotalNet - totals.Net
	tax := totals.Gross - totals.Net
	effectiveTaxRate := 0.0
	if totals.Net > 0 {
		effectiveTaxRate = tax / totals.Net * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"event_days":         eventDays,
		"subtotal":           roundAnalyticsMoney(subtotalNet),
		"global_discount":    math.Round(globalDiscount*100) / 100,
		"netto":              roundAnalyticsMoney(totals.Net),
		"tax_rate":           math.Round(effectiveTaxRate*100) / 100,
		"tax":                math.Round(tax*100) / 100,
		"brutto":             roundAnalyticsMoney(totals.Gross),
		"multiply_by_days":   job.MultiplyByDays,
		"prices_include_tax": job.PricesIncludeTax,
	})
}

type PriceSettingsInput struct {
	MultiplyByDays   *bool `json:"multiply_by_days"`
	PricesIncludeTax *bool `json:"prices_include_tax"`
}

func (h *PositionHandler) UpdatePriceSettings(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var input PriceSettingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.MultiplyByDays != nil {
		updates["multiply_by_days"] = *input.MultiplyByDays
	}
	if input.PricesIncludeTax != nil {
		updates["prices_include_tax"] = *input.PricesIncludeTax
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields provided"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var currentMultiplyByDays bool
		result := tx.Raw("SELECT multiply_by_days FROM jobs WHERE jobid = ? FOR UPDATE", uint(jobID)).Scan(&currentMultiplyByDays)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		result = tx.Model(&models.Job{}).Where("jobid = ?", uint(jobID)).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if input.MultiplyByDays != nil && currentMultiplyByDays != *input.MultiplyByDays {
			if err := tx.Exec(`
				UPDATE job_rental_equipment AS jre
				SET total_cost = ROUND((
					jre.total_cost * CASE
						WHEN ? THEN GREATEST(jre.days_used, 1)::numeric
						ELSE 1.0 / GREATEST(jre.days_used, 1)::numeric
					END
				)::numeric, 2),
				updated_at = NOW()
				WHERE jre.job_id = ?`, *input.MultiplyByDays, uint(jobID)).Error; err != nil {
				return err
			}
		}
		return syncJobRevenueIfPositions(tx, uint(jobID))
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func calcEventDays(start, end *time.Time) int {
	if start == nil || end == nil {
		return 1
	}
	days := int(end.Sub(*start).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

// GetRentalCatalog returns all active rental equipment items for selection in job positions.
// Uses raw column names matching the actual PostgreSQL schema (id, name, supplier)
// rather than the GORM model tags which reference a different legacy schema.
func (h *PositionHandler) GetRentalCatalog(c *gin.Context) {
	type catalogItem struct {
		EquipmentID  uint    `json:"equipmentID"`
		ProductName  string  `json:"productName"`
		SupplierName string  `json:"supplierName"`
		RentalPrice  float64 `json:"rentalPrice"`
		Category     string  `json:"category"`
	}
	var items []catalogItem
	if err := h.db.Table("rental_equipment").
		Where("is_active = ?", true).
		Order("name ASC").
		Select("id AS equipment_id, name AS product_name, supplier AS supplier_name, rental_price, category").
		Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
