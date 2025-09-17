package repository

import (
	"fmt"
	"strings"
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *Database
}

func NewJobRepository(db *Database) *JobRepository {
	return &JobRepository{db: db}
}

// loadProductsForJobDevices manually loads products for job devices
// This is a workaround for GORM nested preloading issues
func (r *JobRepository) loadProductsForJobDevices(jobDevices []models.JobDevice) {
	for i := range jobDevices {
		jd := &jobDevices[i]
		if jd.Device.ProductID != nil {
			var product models.Product
			productErr := r.db.Where("productID = ?", *jd.Device.ProductID).First(&product).Error
			if productErr == nil {
				jd.Device.Product = &product
			}
		}
	}
}

func (r *JobRepository) Create(job *models.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetByID(id uint) (*models.Job, error) {
	var job models.Job
	err := r.db.Preload("JobDevices.Device").First(&job, id).Error
	if err != nil {
		fmt.Printf("🔧 DEBUG JobRepo.GetByID: Error loading job %d: %v\n", id, err)
		return nil, err
	}
	
	// Manually load Customer
	if job.CustomerID > 0 {
		var customer models.Customer
		if err := r.db.Where("customerID = ?", job.CustomerID).First(&customer).Error; err != nil {
			fmt.Printf("🔧 DEBUG JobRepo.GetByID: Failed to load customer %d: %v\n", job.CustomerID, err)
		} else {
			job.Customer = customer
			fmt.Printf("🔧 DEBUG JobRepo.GetByID: Loaded customer %d: %s\n", customer.CustomerID, 
				func() string {
					if customer.CompanyName != nil && *customer.CompanyName != "" {
						return *customer.CompanyName
					}
					if customer.FirstName != nil && customer.LastName != nil {
						return *customer.FirstName + " " + *customer.LastName
					}
					return "No Name"
				}())
		}
	}
	
	// Manually load Status
	if job.StatusID > 0 {
		var status models.Status
		if err := r.db.Where("statusID = ?", job.StatusID).First(&status).Error; err != nil {
			fmt.Printf("🔧 DEBUG JobRepo.GetByID: Failed to load status %d: %v\n", job.StatusID, err)
		} else {
			job.Status = status
			fmt.Printf("🔧 DEBUG JobRepo.GetByID: Loaded status %d: %s\n", status.StatusID, status.Status)
		}
	}
	
	// Add device count
	var deviceCount int64
	if err := r.db.DB.Table("jobdevices").Where("jobID = ?", job.JobID).Count(&deviceCount).Error; err != nil {
		deviceCount = 0
	}
	job.DeviceCount = int(deviceCount)
	
	// Manually load products for each device
	r.loadProductsForJobDevices(job.JobDevices)
	
	fmt.Printf("🔧 DEBUG JobRepo.GetByID: Loaded job %d with description: '%s'\n", id, func() string {
		if job.Description == nil {
			return "<nil>"
		}
		return *job.Description
	}())
	
	return &job, nil
}

func (r *JobRepository) Update(job *models.Job) error {
	fmt.Printf("🔧 DEBUG JobRepo.Update: Saving job ID %d with description: '%s'\n", job.JobID, func() string {
		if job.Description == nil {
			return "<nil>"
		}
		return *job.Description
	}())
	
	// Use Updates instead of Save to ensure all fields are updated
	result := r.db.Model(job).Where("jobID = ?", job.JobID).Updates(map[string]interface{}{
		"customerID":     job.CustomerID,
		"statusID":       job.StatusID,
		"description":    job.Description,
		"startDate":      job.StartDate,
		"endDate":        job.EndDate,
		"revenue":        job.Revenue,
		"discount":       job.Discount,
		"discount_type":  job.DiscountType,
		"jobcategoryID":  job.JobCategoryID,
		"final_revenue":  job.FinalRevenue,
	})
	
	if result.Error != nil {
		fmt.Printf("🔧 DEBUG JobRepo.Update: Error: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Printf("🔧 DEBUG JobRepo.Update: Success! Rows affected: %d\n", result.RowsAffected)
	
	// Verify the update by reading the job back from DB
	var verifyJob models.Job
	verifyResult := r.db.Where("jobID = ?", job.JobID).First(&verifyJob)
	if verifyResult.Error == nil {
		fmt.Printf("🔧 DEBUG JobRepo.Update: Verification - DB now has description: '%s'\n", func() string {
			if verifyJob.Description == nil {
				return "<nil>"
			}
			return *verifyJob.Description
		}())
	} else {
		fmt.Printf("🔧 DEBUG JobRepo.Update: Verification failed: %v\n", verifyResult.Error)
	}
	
	return nil
}

// RemoveAllDevicesFromJob removes all devices assigned to a specific job
func (r *JobRepository) RemoveAllDevicesFromJob(jobID uint) error {
	return r.db.Where("jobID = ?", jobID).Delete(&models.JobDevice{}).Error
}

func (r *JobRepository) Delete(id uint) error {
	// Start a transaction to ensure all deletions succeed or fail together
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	// First, remove all devices from the job to avoid foreign key constraint issues
	if err := tx.Where("jobID = ?", id).Delete(&models.JobDevice{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove devices from job: %v", err)
	}
	
	// Second, remove all employee-job assignments
	if err := tx.Exec("DELETE FROM employeejob WHERE jobID = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove employee assignments from job: %v", err)
	}
	
	// Then delete the job itself
	if err := tx.Delete(&models.Job{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Commit the transaction
	return tx.Commit().Error
}

func (r *JobRepository) List(params *models.FilterParams) ([]models.JobWithDetails, error) {
	var jobs []models.JobWithDetails

	var sqlQuery string
	var args []interface{}

	sqlQuery = `SELECT j.jobID, j.customerID, j.statusID, 
			j.description, j.startDate, j.endDate, 
			j.revenue, j.final_revenue,
			CONCAT(COALESCE(c.companyname, ''), ' ', COALESCE(c.firstname, ''), ' ', COALESCE(c.lastname, '')) as customer_name, 
			s.status as status_name,
			COUNT(DISTINCT jd.deviceID) as device_count,
			COALESCE(j.final_revenue, j.revenue) as total_revenue
		FROM jobs j 
		LEFT JOIN customers c ON j.customerID = c.customerID
		LEFT JOIN status s ON j.statusID = s.statusID
		LEFT JOIN jobdevices jd ON j.jobID = jd.jobID`

	// Build WHERE conditions
	var conditions []string

	if params.StartDate != nil {
		conditions = append(conditions, "j.startDate >= ?")
		args = append(args, *params.StartDate)
	}
	if params.EndDate != nil {
		conditions = append(conditions, "j.endDate <= ?")
		args = append(args, *params.EndDate)
	}
	if params.CustomerID != nil {
		conditions = append(conditions, "j.customerID = ?")
		args = append(args, *params.CustomerID)
	}
	if params.StatusID != nil {
		conditions = append(conditions, "j.statusID = ?")
		args = append(args, *params.StatusID)
	}
	if params.MinRevenue != nil {
		conditions = append(conditions, "j.revenue >= ?")
		args = append(args, *params.MinRevenue)
	}
	if params.MaxRevenue != nil {
		conditions = append(conditions, "j.revenue <= ?")
		args = append(args, *params.MaxRevenue)
	}
	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		conditions = append(conditions, "(j.description LIKE ? OR c.companyname LIKE ? OR c.firstname LIKE ? OR c.lastname LIKE ?)")
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Add WHERE clause if conditions exist
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " GROUP BY j.jobID, j.customerID, j.statusID, j.description, j.startDate, j.endDate, j.revenue, j.final_revenue, customer_name, s.status"

	// Add ORDER BY
	sqlQuery += " ORDER BY j.jobID DESC"

	// Add pagination
	if params.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", params.Limit)
	}
	if params.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", params.Offset)
	}

	err := r.db.Raw(sqlQuery, args...).Scan(&jobs).Error
	return jobs, err
}

func (r *JobRepository) GetJobDevices(jobID uint) ([]models.JobDevice, error) {
	var jobDevices []models.JobDevice
	
	// Load JobDevices with Device, then manually preload Products
	err := r.db.Where("jobID = ?", jobID).
		Preload("Device").
		Find(&jobDevices).Error
	
	if err != nil {
		return nil, err
	}
	
	// Manually load products for each device to ensure they're loaded correctly
	r.loadProductsForJobDevices(jobDevices)
	
	return jobDevices, err
}

func (r *JobRepository) AssignDevice(jobID uint, deviceID string, price float64) error {
	fmt.Printf("🚨 DEBUG: NEW AssignDevice called! jobID=%d, deviceID=%s\n", jobID, deviceID)
	
	// Get the job to check its date range
	var job models.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return fmt.Errorf("job not found: %v", err)
	}

	fmt.Printf("🚨 DEBUG: Job %d dates: %v to %v\n", jobID, job.StartDate, job.EndDate)

	// Check if device is available for this job's date range
	// Implement the date-based availability check directly
	
	// Check if device is already assigned to this specific job
	var existingAssignment models.JobDevice
	err = r.db.Where("deviceID = ? AND jobID = ?", deviceID, jobID).First(&existingAssignment).Error
	if err == nil {
		return fmt.Errorf("device is already assigned to this job")
	}

	// Check for conflicting assignments based on date overlap
	if job.StartDate != nil && job.EndDate != nil {
		var conflictingJob models.JobDevice
		err = r.db.Joins("JOIN jobs ON jobdevices.jobID = jobs.jobID").
			Where(`jobdevices.deviceID = ? 
				AND jobs.jobID != ? 
				AND jobs.startDate <= ? 
				AND jobs.endDate >= ? 
				AND jobs.statusID IN (
					SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
				)`, deviceID, jobID, job.EndDate, job.StartDate).
			First(&conflictingJob).Error
		
		if err == nil {
			// Get conflicting job details for error message
			var conflictJob models.Job
			r.db.Where("jobID = ?", conflictingJob.JobID).First(&conflictJob)
			return fmt.Errorf("device is already assigned to job %d (dates: %s to %s)", 
				conflictJob.JobID, 
				conflictJob.StartDate.Format("2006-01-02"), 
				conflictJob.EndDate.Format("2006-01-02"))
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking device availability: %v", err)
		}
	} else {
		// If no dates specified, fall back to simple assignment check
		err = r.db.Where("deviceID = ?", deviceID).First(&existingAssignment).Error
		if err == nil {
			return fmt.Errorf("device is already assigned to job %d", existingAssignment.JobID)
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}

	// Create new assignment
	jobDevice := &models.JobDevice{
		JobID:    jobID,
		DeviceID: deviceID,
	}

	// Only set custom price if it's greater than 0
	if price > 0 {
		jobDevice.CustomPrice = &price
	}

	err = r.db.Create(jobDevice).Error
	if err != nil {
		return err
	}

	// Recalculate and update job revenue
	return r.CalculateAndUpdateRevenue(jobID)
}

func (r *JobRepository) RemoveDevice(jobID uint, deviceID string) error {
	err := r.db.Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Delete(&models.JobDevice{}).Error
	if err != nil {
		return err
	}

	// Recalculate and update job revenue
	return r.CalculateAndUpdateRevenue(jobID)
}

func (r *JobRepository) UnassignDevice(jobID uint, deviceID string) error {
	// Remove device from job
	err := r.db.Where("jobID = ? AND deviceID = ?", jobID, deviceID).Delete(&models.JobDevice{}).Error
	if err != nil {
		return fmt.Errorf("failed to unassign device %s from job %d: %v", deviceID, jobID, err)
	}
	
	// Update device status to free
	err = r.db.Model(&models.Device{}).Where("deviceID = ?", deviceID).Update("status", "free").Error
	if err != nil {
		return fmt.Errorf("failed to update device status: %v", err)
	}
	
	// Recalculate and update job revenue
	return r.CalculateAndUpdateRevenue(jobID)
}

func (r *JobRepository) BulkAssignDevices(jobID uint, deviceIDs []string, price float64) ([]models.ScanResult, error) {
	var results []models.ScanResult
	hasSuccessfulAssignments := false

	for _, deviceID := range deviceIDs {
		result := models.ScanResult{
			DeviceID: deviceID,
		}

		// Find device by serial number or device ID
		var device models.Device
		err := r.db.Where("serialnumber = ? OR deviceID = ?", deviceID, deviceID).First(&device).Error
		if err != nil {
			result.Success = false
			result.Message = "Device not found"
			results = append(results, result)
			continue
		}

		// Try to assign device (without triggering revenue calculation yet)
		err = r.assignDeviceWithoutRevenue(jobID, device.DeviceID, price)
		if err != nil {
			result.Success = false
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = "Device assigned successfully"
			result.Device = &device
			hasSuccessfulAssignments = true
		}

		results = append(results, result)
	}

	// Calculate revenue once at the end for efficiency
	if hasSuccessfulAssignments {
		r.CalculateAndUpdateRevenue(jobID)
	}

	return results, nil
}

// Helper method to assign device without triggering revenue calculation
func (r *JobRepository) assignDeviceWithoutRevenue(jobID uint, deviceID string, price float64) error {
	// Get the job to check its date range
	var job models.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return fmt.Errorf("job not found: %v", err)
	}

	// Check if device is already assigned to this specific job
	var existingAssignment models.JobDevice
	err = r.db.Where("deviceID = ? AND jobID = ?", deviceID, jobID).First(&existingAssignment).Error
	if err == nil {
		return fmt.Errorf("device is already assigned to this job")
	}

	// Check for conflicting assignments based on date overlap
	if job.StartDate != nil && job.EndDate != nil {
		var conflictingJob models.JobDevice
		err = r.db.Joins("JOIN jobs ON jobdevices.jobID = jobs.jobID").
			Where(`jobdevices.deviceID = ? 
				AND jobs.jobID != ? 
				AND jobs.startDate <= ? 
				AND jobs.endDate >= ? 
				AND jobs.statusID IN (
					SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
				)`, deviceID, jobID, job.EndDate, job.StartDate).
			First(&conflictingJob).Error
		
		if err == nil {
			var conflictJob models.Job
			r.db.Where("jobID = ?", conflictingJob.JobID).First(&conflictJob)
			return fmt.Errorf("device is already assigned to job %d (dates: %s to %s)", 
				conflictJob.JobID, 
				conflictJob.StartDate.Format("2006-01-02"), 
				conflictJob.EndDate.Format("2006-01-02"))
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("error checking device availability: %v", err)
		}
	} else {
		// If no dates specified, fall back to simple assignment check
		err = r.db.Where("deviceID = ?", deviceID).First(&existingAssignment).Error
		if err == nil {
			return fmt.Errorf("device is already assigned to job %d", existingAssignment.JobID)
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}

	// Create new assignment
	jobDevice := &models.JobDevice{
		JobID:    jobID,
		DeviceID: deviceID,
	}

	// Only set custom price if it's greater than 0
	if price > 0 {
		jobDevice.CustomPrice = &price
	}

	return r.db.Create(jobDevice).Error
}

func (r *JobRepository) GetJobStats(jobID uint) (*models.JobWithDetails, error) {
	var job models.JobWithDetails
	err := r.db.Table("jobs j").
		Select(`j.*, c.name as customer_name, s.name as status_name,
				COUNT(DISTINCT jd.device_id) as device_count,
				COALESCE(SUM(jd.price), 0) as total_revenue`).
		Joins("LEFT JOIN customers c ON j.customer_id = c.id").
		Joins("LEFT JOIN statuses s ON j.status_id = s.id").
		Joins("LEFT JOIN job_devices jd ON j.id = jd.job_id AND jd.removed_at IS NULL").
		Where("j.id = ?", jobID).
		Group("j.id").
		First(&job).Error

	return &job, err
}

func (r *JobRepository) CalculateAndUpdateRevenue(jobID uint) error {
	// Get the job with dates
	var job models.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return err
	}

	// Revenue is calculated as flat rates, not per day

	// Calculate total revenue from job devices
	var totalRevenue float64
	var jobDevices []models.JobDevice
	err = r.db.Where("jobID = ?", jobID).
		Preload("Device").
		Find(&jobDevices).Error
	if err != nil {
		return err
	}
	
	// Manually load products for each device
	r.loadProductsForJobDevices(jobDevices)

	for _, jd := range jobDevices {
		if jd.CustomPrice != nil && *jd.CustomPrice > 0 {
			// Use custom price as-is (flat rate, not per day)
			totalRevenue += *jd.CustomPrice
		} else if jd.Device.Product != nil && jd.Device.Product.ItemCostPerDay != nil {
			// Use product price as flat rate (not per day)
			totalRevenue += *jd.Device.Product.ItemCostPerDay
		}
	}

	// Update the job revenue
	job.Revenue = totalRevenue
	
	// Calculate final revenue after discount
	var finalRevenue float64
	if job.DiscountType == "percent" {
		// Percentage discount
		finalRevenue = totalRevenue * (1 - job.Discount/100)
	} else {
		// Fixed amount discount
		finalRevenue = totalRevenue - job.Discount
		if finalRevenue < 0 {
			finalRevenue = 0 // Cannot be negative
		}
	}
	job.FinalRevenue = &finalRevenue
	
	return r.db.Save(&job).Error
}

func (r *JobRepository) UpdateFinalRevenue(jobID uint) error {
	// Get the job with current revenue
	var job models.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return err
	}

	// Calculate final revenue after discount using existing revenue
	var finalRevenue float64
	if job.DiscountType == "percent" {
		// Percentage discount
		finalRevenue = job.Revenue * (1 - job.Discount/100)
	} else {
		// Fixed amount discount
		finalRevenue = job.Revenue - job.Discount
		if finalRevenue < 0 {
			finalRevenue = 0 // Cannot be negative
		}
	}
	job.FinalRevenue = &finalRevenue
	
	return r.db.Save(&job).Error
}

func (r *JobRepository) UpdateDevicePrice(jobID uint, deviceID string, price float64) error {
	fmt.Printf("🔧 DEBUG UpdateDevicePrice: JobID=%d, DeviceID=%s, Price=%.2f\n", jobID, deviceID, price)
	
	// Update the custom_price for the specific job-device relationship
	// Fix: column name is 'deviceID' not 'device_id'
	result := r.db.Model(&models.JobDevice{}).
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Update("custom_price", price)
	
	fmt.Printf("🔧 DEBUG UpdateDevicePrice: SQL result - Error=%v, RowsAffected=%d\n", result.Error, result.RowsAffected)
	
	if result.Error != nil {
		fmt.Printf("🔧 DEBUG UpdateDevicePrice: Database error: %v\n", result.Error)
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		fmt.Printf("🔧 DEBUG UpdateDevicePrice: No rows affected - device not found\n")
		return fmt.Errorf("device %s not found in job %d", deviceID, jobID)
	}
	
	// Recalculate job revenue after price update
	fmt.Printf("🔧 DEBUG UpdateDevicePrice: Recalculating revenue for job %d\n", jobID)
	err := r.CalculateAndUpdateRevenue(jobID)
	if err != nil {
		fmt.Printf("🔧 DEBUG UpdateDevicePrice: Revenue calculation error: %v\n", err)
		return err
	}
	
	fmt.Printf("🔧 DEBUG UpdateDevicePrice: Success!\n")
	return nil
}

// GetJobDeviceCount returns the total number of devices assigned to a job (performance optimized)
func (r *JobRepository) GetJobDeviceCount(jobID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.JobDevice{}).Where("jobID = ?", jobID).Count(&count).Error
	return int(count), err
}

// ProductSummary represents device count summary by product
type ProductSummary struct {
	ProductName string
	Product     *models.Product
	Count       int
}

// GetJobDeviceProductSummary returns summary of devices grouped by product (ultra-fast)
func (r *JobRepository) GetJobDeviceProductSummary(jobID uint) ([]ProductSummary, error) {
	var summaries []ProductSummary

	// Ultra-fast query with minimal JOINs and optimized for performance
	rows, err := r.db.Raw(`
		SELECT
			COALESCE(p.name, 'Unknown Product') as product_name,
			COUNT(*) as count,
			p.productID,
			p.itemcostperday
		FROM jobdevices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		WHERE jd.jobID = ?
		GROUP BY p.productID, p.name, p.itemcostperday
		ORDER BY count DESC, p.name
	`, jobID).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var productName string
		var count int
		var productID *uint
		var itemCostPerDay *float64

		if err := rows.Scan(&productName, &count, &productID, &itemCostPerDay); err != nil {
			return nil, err
		}

		// Create lightweight product object without additional DB query
		var product *models.Product
		if productID != nil {
			product = &models.Product{
				ProductID:      *productID,
				Name:           productName,
				ItemCostPerDay: itemCostPerDay,
			}
		}

		summaries = append(summaries, ProductSummary{
			ProductName: productName,
			Product:     product,
			Count:       count,
		})
	}

	return summaries, nil
}

// GetJobDevicesPaginated returns devices for a job with pagination
func (r *JobRepository) GetJobDevicesPaginated(jobID uint, productName string, page int, limit int) ([]models.JobDevice, error) {
	var jobDevices []models.JobDevice
	offset := (page - 1) * limit

	query := r.db.Where("jobID = ?", jobID)

	// Filter by product if specified
	if productName != "" && productName != "Unknown Product" {
		query = query.Joins("JOIN devices d ON jobdevices.deviceID = d.deviceID").
			Joins("JOIN products p ON d.productID = p.productID").
			Where("p.name = ?", productName)
	} else if productName == "Unknown Product" {
		query = query.Joins("LEFT JOIN devices d ON jobdevices.deviceID = d.deviceID").
			Where("d.productID IS NULL")
	}

	err := query.Preload("Device").
		Limit(limit).
		Offset(offset).
		Find(&jobDevices).Error

	if err != nil {
		return nil, err
	}

	// Manually load products for each device
	r.loadProductsForJobDevices(jobDevices)

	return jobDevices, nil
}