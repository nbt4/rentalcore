package repository

import (
	"fmt"
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type RentalEquipmentRepository struct {
	db *Database
}

func NewRentalEquipmentRepository(db *Database) *RentalEquipmentRepository {
	return &RentalEquipmentRepository{db: db}
}

// GetAllRentalEquipment returns all rental equipment items
func (r *RentalEquipmentRepository) GetAllRentalEquipment(rentalEquipment *[]models.RentalEquipment) error {
	return r.db.Find(rentalEquipment).Error
}

// GetRentalEquipmentByID returns a specific rental equipment item by ID
func (r *RentalEquipmentRepository) GetRentalEquipmentByID(equipmentID uint, rentalEquipment *models.RentalEquipment) error {
	return r.db.First(rentalEquipment, equipmentID).Error
}

// CreateRentalEquipment creates a new rental equipment item
func (r *RentalEquipmentRepository) CreateRentalEquipment(rentalEquipment *models.RentalEquipment) error {
	return r.db.Create(rentalEquipment).Error
}

// UpdateRentalEquipment updates an existing rental equipment item
func (r *RentalEquipmentRepository) UpdateRentalEquipment(rentalEquipment *models.RentalEquipment) error {
	return r.db.Save(rentalEquipment).Error
}

// DeleteRentalEquipment deletes a rental equipment item
func (r *RentalEquipmentRepository) DeleteRentalEquipment(equipmentID uint) error {
	// First check if the equipment is used in any jobs
	var count int64
	err := r.db.Model(&models.JobRentalEquipment{}).Where("equipment_id = ?", equipmentID).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check equipment usage: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete equipment that is used in %d job(s)", count)
	}

	return r.db.Delete(&models.RentalEquipment{}, equipmentID).Error
}

// AddRentalToJob adds rental equipment to a job
func (r *RentalEquipmentRepository) AddRentalToJob(jobRental *models.JobRentalEquipment) error {
	// Get rental equipment to calculate total cost
	var equipment models.RentalEquipment
	if err := r.db.First(&equipment, jobRental.EquipmentID).Error; err != nil {
		return fmt.Errorf("rental equipment not found: %v", err)
	}

	// Calculate total cost
	jobRental.TotalCost = equipment.RentalPrice * float64(jobRental.Quantity) * float64(jobRental.DaysUsed)

	// Check if already exists, then update or create
	var existingRental models.JobRentalEquipment
	err := r.db.Where("job_id = ? AND equipment_id = ?", jobRental.JobID, jobRental.EquipmentID).First(&existingRental).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		return r.db.Create(jobRental).Error
	} else if err != nil {
		return err
	} else {
		// Update existing
		existingRental.Quantity = jobRental.Quantity
		existingRental.DaysUsed = jobRental.DaysUsed
		existingRental.TotalCost = jobRental.TotalCost
		existingRental.Notes = jobRental.Notes
		return r.db.Save(&existingRental).Error
	}
}

// CreateRentalEquipmentFromManualEntry creates rental equipment and adds it to job in one transaction
func (r *RentalEquipmentRepository) CreateRentalEquipmentFromManualEntry(request *models.ManualRentalEntryRequest, createdBy *uint) (*models.RentalEquipment, *models.JobRentalEquipment, error) {
	var rentalEquipment *models.RentalEquipment
	var jobRental *models.JobRentalEquipment

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Create rental equipment
		rentalEquipment = &models.RentalEquipment{
			ProductName:  request.ProductName,
			SupplierName: request.SupplierName,
			RentalPrice:  request.RentalPrice,
			Category:     request.Category,
			Description:  request.Description,
			Notes:        request.Notes,
			IsActive:     true,
			CreatedBy:    createdBy,
		}

		if err := tx.Create(rentalEquipment).Error; err != nil {
			return fmt.Errorf("failed to create rental equipment: %v", err)
		}

		// Add to job
		totalCost := request.RentalPrice * float64(request.Quantity) * float64(request.DaysUsed)

		jobRental = &models.JobRentalEquipment{
			JobID:       request.JobID,
			EquipmentID: rentalEquipment.EquipmentID,
			Quantity:    request.Quantity,
			DaysUsed:    request.DaysUsed,
			TotalCost:   totalCost,
			Notes:       request.Notes,
		}

		if err := tx.Create(jobRental).Error; err != nil {
			return fmt.Errorf("failed to add rental to job: %v", err)
		}

		return nil
	})

	return rentalEquipment, jobRental, err
}

// GetJobRentalEquipment returns all rental equipment for a specific job
func (r *RentalEquipmentRepository) GetJobRentalEquipment(jobID uint, jobRentals *[]models.JobRentalEquipment) error {
	return r.db.Preload("RentalEquipment").Where("job_id = ?", jobID).Find(jobRentals).Error
}

// RemoveRentalFromJob removes rental equipment from a job
func (r *RentalEquipmentRepository) RemoveRentalFromJob(jobID, equipmentID uint) error {
	return r.db.Where("job_id = ? AND equipment_id = ?", jobID, equipmentID).Delete(&models.JobRentalEquipment{}).Error
}

// GetRentalEquipmentAnalytics returns analytics data for rental equipment
func (r *RentalEquipmentRepository) GetRentalEquipmentAnalytics() (*models.RentalEquipmentAnalytics, error) {
	analytics := &models.RentalEquipmentAnalytics{}

	// Basic counts
	var totalCount, activeCount int64
	r.db.Model(&models.RentalEquipment{}).Count(&totalCount)
	r.db.Model(&models.RentalEquipment{}).Where("is_active = ?", true).Count(&activeCount)
	analytics.TotalEquipmentItems = int(totalCount)
	analytics.ActiveEquipmentItems = int(activeCount)

	// Count distinct suppliers
	var suppliers []string
	r.db.Model(&models.RentalEquipment{}).Distinct("supplier_name").Pluck("supplier_name", &suppliers)
	analytics.TotalSuppliersCount = len(suppliers)

	// Total rental revenue
	r.db.Model(&models.JobRentalEquipment{}).Select("COALESCE(SUM(total_cost), 0)").Scan(&analytics.TotalRentalRevenue)

	// Basic category breakdown (simplified for now)
	var categories []models.RentalCategoryBreakdown

	// Get categories with equipment count
	type CategorySummary struct {
		Category       string
		EquipmentCount int64
		TotalRevenue   float64
	}

	var categorySummaries []CategorySummary
	r.db.Model(&models.RentalEquipment{}).
		Select("COALESCE(category, 'Uncategorized') as category, COUNT(*) as equipment_count").
		Group("category").
		Find(&categorySummaries)

	for _, summary := range categorySummaries {
		var avgRevenue float64
		if summary.EquipmentCount > 0 {
			avgRevenue = summary.TotalRevenue / float64(summary.EquipmentCount)
		}

		categories = append(categories, models.RentalCategoryBreakdown{
			Category:               summary.Category,
			EquipmentCount:         int(summary.EquipmentCount),
			TotalRevenue:          summary.TotalRevenue,
			UsageCount:            0, // Simplified for now
			AvgRevenuePerEquipment: avgRevenue,
		})
	}

	// For simplicity, using simplified data for now
	analytics.MostUsedEquipment = []models.MostUsedRentalEquipment{}
	analytics.TopSuppliers = []models.TopRentalSupplier{}
	analytics.CategoryBreakdown = categories
	analytics.MonthlyRentalRevenue = []models.MonthlyRentalRevenue{}

	return analytics, nil
}