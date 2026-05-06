package repository

import (
	"fmt"
	"go-barcode-webapp/internal/models"
	"log"
	"strings"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *Database
}

const jobRepoDebugLogsEnabled = false

func jobRepoDebugLog(format string, args ...interface{}) {
	if !jobRepoDebugLogsEnabled {
		return
	}
	fmt.Printf(format, args...)
}

func computeFinalRevenue(revenue, discount float64, discountType string) float64 {
	finalRevenue := revenue

	switch strings.ToLower(discountType) {
	case "percent", "percentage":
		finalRevenue = revenue * (1 - discount/100)
	default:
		finalRevenue = revenue - discount
	}

	if finalRevenue < 0 {
		finalRevenue = 0
	}

	return finalRevenue
}

func NewJobRepository(db *Database) *JobRepository {
	return &JobRepository{db: db}
}

// GetDB returns the underlying database connection
func (r *JobRepository) GetDB() *Database {
	return r.db
}

// loadProductsForJobDevices manually loads products for job devices
// This is a workaround for GORM nested preloading issues
func (r *JobRepository) loadProductsForJobDevices(jobDevices []models.JobDevice) {
	productIDs := make([]uint, 0, len(jobDevices))
	uniqueIDs := make(map[uint]struct{})

	for i := range jobDevices {
		productIDPtr := jobDevices[i].Device.ProductID
		if productIDPtr == nil {
			continue
		}

		productID := *productIDPtr
		if _, exists := uniqueIDs[productID]; exists {
			continue
		}

		uniqueIDs[productID] = struct{}{}
		productIDs = append(productIDs, productID)
	}

	if len(productIDs) == 0 {
		return
	}

	var products []models.Product
	if err := r.db.Where("productID IN ?", productIDs).Find(&products).Error; err != nil {
		log.Printf("Warning: failed to preload products for job devices: %v", err)
		return
	}

	productMap := make(map[uint]*models.Product, len(products))
	for i := range products {
		product := &products[i]
		productMap[product.ProductID] = product
	}

	for i := range jobDevices {
		jd := &jobDevices[i]
		if jd.Device.ProductID == nil {
			continue
		}

		if product, ok := productMap[*jd.Device.ProductID]; ok {
			jd.Device.Product = product
		}
	}
}

func (r *JobRepository) Create(job *models.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetByID(id uint) (*models.Job, error) {
	var job models.Job
	err := r.db.
		Preload("JobDevices.Device").
		Preload("Creator").
		First(&job, id).Error
	if err != nil {
		jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Error loading job %d: %v\n", id, err)
		return nil, err
	}

	// Manually load Customer
	if job.CustomerID > 0 {
		var customer models.Customer
		if err := r.db.Where("customerID = ?", job.CustomerID).First(&customer).Error; err != nil {
			jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Failed to load customer %d: %v\n", job.CustomerID, err)
		} else {
			job.Customer = customer
			jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Loaded customer %d: %s\n", customer.CustomerID,
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
			jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Failed to load status %d: %v\n", job.StatusID, err)
		} else {
			job.Status = status
			jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Loaded status %d: %s\n", status.StatusID, status.Status)
		}
	}

	// Add device count
	var deviceCount int64
	if err := r.db.DB.Table("job_devices").Where("jobID = ?", job.JobID).Count(&deviceCount).Error; err != nil {
		deviceCount = 0
	}
	job.DeviceCount = int(deviceCount)

	// Manually load products for each device
	r.loadProductsForJobDevices(job.JobDevices)

	jobRepoDebugLog("🔧 DEBUG JobRepo.GetByID: Loaded job %d with description: '%s'\n", id, func() string {
		if job.Description == nil {
			return "<nil>"
		}
		return *job.Description
	}())

	return &job, nil
}

func (r *JobRepository) Update(job *models.Job) error {
	jobRepoDebugLog("🔧 DEBUG JobRepo.Update: Saving job ID %d with description: '%s'\n", job.JobID, func() string {
		if job.Description == nil {
			return "<nil>"
		}
		return *job.Description
	}())

	finalRevenue := computeFinalRevenue(job.Revenue, job.Discount, job.DiscountType)
	job.FinalRevenue = &finalRevenue

	// Use Updates instead of Save to ensure all fields are updated
	result := r.db.Model(job).Where("jobID = ?", job.JobID).Updates(map[string]interface{}{
		"customerid":    job.CustomerID,
		"statusid":      job.StatusID,
		"description":   job.Description,
		"startdate":     job.StartDate,
		"enddate":       job.EndDate,
		"revenue":       job.Revenue,
		"discount":      job.Discount,
		"discount_type": job.DiscountType,
		"jobcategoryid": job.JobCategoryID,
		"final_revenue": finalRevenue,
	})

	if result.Error != nil {
		jobRepoDebugLog("🔧 DEBUG JobRepo.Update: Error: %v\n", result.Error)
		return result.Error
	}

	jobRepoDebugLog("🔧 DEBUG JobRepo.Update: Success! Rows affected: %d\n", result.RowsAffected)

	// Verify the update by reading the job back from DB
	var verifyJob models.Job
	verifyResult := r.db.Where("jobID = ?", job.JobID).First(&verifyJob)
	if verifyResult.Error == nil {
		jobRepoDebugLog("🔧 DEBUG JobRepo.Update: Verification - DB now has description: '%s'\n", func() string {
			if verifyJob.Description == nil {
				return "<nil>"
			}
			return *verifyJob.Description
		}())
	} else {
		jobRepoDebugLog("🔧 DEBUG JobRepo.Update: Verification failed: %v\n", verifyResult.Error)
	}

	return nil
}

// Exists returns true if a job with the provided ID exists.
func (r *JobRepository) Exists(jobID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Job{}).Where("jobID = ?", jobID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetProductName returns the product name for the given productID.
func (r *JobRepository) GetProductName(productID uint) (string, error) {
	if productID == 0 {
		return "", nil
	}

	var product models.Product
	if err := r.db.Select("name").First(&product, productID).Error; err != nil {
		return "", err
	}
	return product.Name, nil
}

// RemoveAllDevicesFromJob removes all devices assigned to a specific job
func (r *JobRepository) RemoveAllDevicesFromJob(jobID uint) error {
	return r.db.Where("jobID = ?", jobID).Delete(&models.JobDevice{}).Error
}

func (r *JobRepository) Delete(id uint) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Collect device IDs before removing them, so we can reset their status
	var deviceIDs []string
	tx.Model(&models.JobDevice{}).Where("jobID = ?", id).Pluck("deviceID", &deviceIDs)

	// Reset on_job devices back to in_storage
	if len(deviceIDs) > 0 {
		if err := tx.Model(&models.Device{}).
			Where("deviceID IN ? AND status = ?", deviceIDs, "on_job").
			Update("status", "in_storage").Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to reset device statuses: %v", err)
		}
	}

	// Remove all job_devices rows
	if err := tx.Where("jobID = ?", id).Delete(&models.JobDevice{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove devices from job: %v", err)
	}

	// Delete the job itself
	if err := tx.Delete(&models.Job{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *JobRepository) List(params *models.FilterParams) ([]models.JobWithDetails, error) {
	var jobs []models.JobWithDetails

	var sqlQuery string
	var args []interface{}

	sqlQuery = `SELECT j.jobid, j.job_code, j.customerid, j.statusid, j.jobcategoryid,
			j.description, j.startdate, j.enddate,
			j.revenue, j.final_revenue,
			CONCAT(COALESCE(c.companyname, ''), ' ', COALESCE(c.firstname, ''), ' ', COALESCE(c.lastname, '')) as customer_name,
			s.status as status_name,
			jc.name as category_name,
			COUNT(DISTINCT jd.deviceid) as device_count,
			COALESCE(j.final_revenue, j.revenue) as total_revenue
		FROM jobs j
		LEFT JOIN customers c ON j.customerid = c.customerid
		LEFT JOIN status s ON j.statusid = s.statusid
		LEFT JOIN jobCategory jc ON j.jobcategoryid = jc.jobcategoryid
		LEFT JOIN job_devices jd ON j.jobid = jd.jobid`

	// Build WHERE conditions
	var conditions []string

	if params.StartDate != nil {
		conditions = append(conditions, "j.startdate >= ?")
		args = append(args, *params.StartDate)
	}
	if params.EndDate != nil {
		conditions = append(conditions, "j.enddate <= ?")
		args = append(args, *params.EndDate)
	}
	if params.CustomerID != nil {
		conditions = append(conditions, "j.customerid = ?")
		args = append(args, *params.CustomerID)
	}
	if params.StatusID != nil {
		conditions = append(conditions, "j.statusid = ?")
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

	sqlQuery += " GROUP BY j.jobid, j.job_code, j.customerid, j.statusid, j.jobcategoryid, j.description, j.startdate, j.enddate, j.revenue, j.final_revenue, customer_name, s.status, category_name"

	// Add ORDER BY
	sqlQuery += " ORDER BY j.jobid DESC"

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
	jobRepoDebugLog("🚨 DEBUG: NEW AssignDevice called! jobID=%d, deviceID=%s\n", jobID, deviceID)

	// Get the job to check its date range
	var job models.Job
	err := r.db.First(&job, jobID).Error
	if err != nil {
		return fmt.Errorf("job not found: %v", err)
	}

	jobRepoDebugLog("🚨 DEBUG: Job %d dates: %v to %v\n", jobID, job.StartDate, job.EndDate)

	// Check if this is a package device
	var device models.Device
	_ = r.db.Preload("Product").Where("deviceID = ?", deviceID).First(&device).Error

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
		err = r.db.Joins("JOIN jobs ON job_devices.jobid = jobs.jobid").
			Where(`job_devices.deviceid = ? 
				AND jobs.jobid != ? 
				AND jobs.startdate <= ? 
				AND jobs.enddate >= ? 
				AND jobs.statusid IN (
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
		JobID:    int(jobID),
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
		err = r.db.Joins("JOIN jobs ON job_devices.jobid = jobs.jobid").
			Where(`job_devices.deviceid = ? 
				AND jobs.jobid != ? 
				AND jobs.startdate <= ? 
				AND jobs.enddate >= ? 
				AND jobs.statusid IN (
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
		JobID:    int(jobID),
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
		if jd.CustomPrice != nil {
			// Use custom price as-is (allow zero for full discounts)
			if *jd.CustomPrice > 0 {
				totalRevenue += *jd.CustomPrice
			}
			continue
		}

		if jd.Device.Product != nil && jd.Device.Product.ItemCostPerDay != nil {
			// Use product price as flat rate (not per day)
			totalRevenue += *jd.Device.Product.ItemCostPerDay
		}
	}

	// Update the job revenue
	job.Revenue = totalRevenue

	// Calculate final revenue after discount
	finalRevenue := computeFinalRevenue(totalRevenue, job.Discount, job.DiscountType)
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
	finalRevenue := computeFinalRevenue(job.Revenue, job.Discount, job.DiscountType)
	job.FinalRevenue = &finalRevenue

	return r.db.Save(&job).Error
}

func (r *JobRepository) UpdateDevicePrice(jobID uint, deviceID string, price float64) error {
	jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: JobID=%d, DeviceID=%s, Price=%.2f\n", jobID, deviceID, price)

	// Update the custom_price for the specific job-device relationship
	// Fix: column name is 'deviceID' not 'device_id'
	result := r.db.Model(&models.JobDevice{}).
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Update("custom_price", price)

	jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: SQL result - Error=%v, RowsAffected=%d\n", result.Error, result.RowsAffected)

	if result.Error != nil {
		jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: Database error: %v\n", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: No rows affected - device not found\n")
		return fmt.Errorf("device %s not found in job %d", deviceID, jobID)
	}

	// Recalculate job revenue after price update
	jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: Recalculating revenue for job %d\n", jobID)
	err := r.CalculateAndUpdateRevenue(jobID)
	if err != nil {
		jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: Revenue calculation error: %v\n", err)
		return err
	}

	jobRepoDebugLog("🔧 DEBUG UpdateDevicePrice: Success!\n")
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
			p.productid,
			p.itemcostperday
		FROM job_devices jd
		LEFT JOIN devices d ON jd.deviceid = d.deviceid
		LEFT JOIN products p ON d.productid = p.productid
		WHERE jd.jobid = ?
		GROUP BY p.productid, p.name, p.itemcostperday
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
		query = query.Joins("JOIN devices d ON job_devices.deviceid = d.deviceid").
			Joins("JOIN products p ON d.productid = p.productid").
			Where("p.name = ?", productName)
	} else if productName == "Unknown Product" {
		query = query.Joins("LEFT JOIN devices d ON job_devices.deviceid = d.deviceid").
			Where("d.productid IS NULL")
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

// handlePackageDeviceAssignment handles the assignment of a package device to a job
// It calculates discounts, adds the package device, and reserves all real devices
func (r *JobRepository) handlePackageDeviceAssignment(jobID uint, packageDeviceID string, job *models.Job, price float64, pkg *models.ProductPackage) error {
	log.Printf("[PACKAGE] Starting package assignment: jobID=%d, packageDeviceID=%s, packageID=%d", jobID, packageDeviceID, pkg.PackageID)

	// Load all items in this package
	var packageItems []models.ProductPackageItem
	if err := r.db.Where("package_id = ?", pkg.PackageID).Find(&packageItems).Error; err != nil {
		return fmt.Errorf("failed to load package items: %w", err)
	}

	log.Printf("[PACKAGE] Found %d items in package %d", len(packageItems), pkg.PackageID)

	// Calculate total regular price of all items
	var regularTotal float64
	for _, item := range packageItems {
		var product models.Product
		if err := r.db.First(&product, item.ProductID).Error; err != nil {
			log.Printf("[PACKAGE] Warning: Could not load product %d: %v", item.ProductID, err)
			continue
		}

		if product.ItemCostPerDay != nil {
			itemPrice := *product.ItemCostPerDay * float64(item.Quantity)
			regularTotal += itemPrice
			log.Printf("[PACKAGE] Product %d (%s): %.2f x %d = %.2f", product.ProductID, product.Name, *product.ItemCostPerDay, item.Quantity, itemPrice)
		}
	}

	// Determine package price (use provided price, or fall back to package.Price)
	packagePrice := price
	if packagePrice == 0 && pkg.Price.Valid {
		packagePrice = pkg.Price.Float64
	}

	// Calculate discount percentage
	var discountPercent float64
	if regularTotal > 0 {
		discountPercent = (regularTotal - packagePrice) / regularTotal
	}

	log.Printf("[PACKAGE] Regular total: %.2f, Package price: %.2f, Discount: %.2f%%", regularTotal, packagePrice, discountPercent*100)

	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// 1. Add the package device itself to job_devices
	packageJobDevice := models.JobDevice{
		JobID:       int(jobID),
		DeviceID:    packageDeviceID,
		CustomPrice: &packagePrice,
		// is_package_item stays false for the package device itself
	}

	if err := tx.Create(&packageJobDevice).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add package device to job: %w", err)
	}

	log.Printf("[PACKAGE] ✓ Added package device %s with price %.2f", packageDeviceID, packagePrice)

	// 2. Reserve real devices for each package item
	for _, item := range packageItems {
		var product models.Product
		if err := tx.First(&product, item.ProductID).Error; err != nil {
			log.Printf("[PACKAGE] Warning: Could not load product %d: %v", item.ProductID, err)
			continue
		}

		// Find available devices for this product
		availableDevices, err := r.findAvailableDevicesForProduct(tx, uint(item.ProductID), item.Quantity, job)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to find available devices for product %d: %w", item.ProductID, err)
		}

		if len(availableDevices) < item.Quantity {
			tx.Rollback()
			return fmt.Errorf("not enough available devices for product %d (%s): need %d, found %d",
				item.ProductID, product.Name, item.Quantity, len(availableDevices))
		}

		// Add the found devices to job_devices with discount applied
		for i := 0; i < item.Quantity; i++ {
			device := availableDevices[i]

			// Calculate discounted price
			var discountedPrice *float64
			if product.ItemCostPerDay != nil {
				price := *product.ItemCostPerDay * (1 - discountPercent)
				discountedPrice = &price
			}

			packageIDInt := pkg.PackageID
			jobDevice := models.JobDevice{
				JobID:         int(jobID),
				DeviceID:      device.DeviceID,
				CustomPrice:   discountedPrice,
				IsPackageItem: true,
				PackageID:     &packageIDInt,
			}

			if err := tx.Create(&jobDevice).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to add device %s to job: %w", device.DeviceID, err)
			}

			log.Printf("[PACKAGE] ✓ Added device %s (product: %s) with discounted price %.2f (is_package_item=true)",
				device.DeviceID, product.Name, *discountedPrice)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit package assignment: %w", err)
	}

	log.Printf("[PACKAGE] ✓ Successfully assigned package %s to job %d", packageDeviceID, jobID)

	// Update revenue calculation
	return r.CalculateAndUpdateRevenue(jobID)
}

// findAvailableDevicesForProduct finds available devices for a given product within a job's date range
func (r *JobRepository) findAvailableDevicesForProduct(tx *gorm.DB, productID uint, quantity int, job *models.Job) ([]models.Device, error) {
	var devices []models.Device

	// Find all devices for this product
	query := tx.Model(&models.Device{}).Where("productID = ?", productID)

	// Exclude devices that are assigned to other jobs with overlapping dates
	if job.StartDate != nil && job.EndDate != nil {
		query = query.Where(`deviceID NOT IN (
			SELECT jd.deviceid
			FROM job_devices jd
			JOIN jobs j ON jd.jobid = j.jobid
			WHERE j.startdate <= ?
			  AND j.enddate >= ?
			  AND j.statusid IN (
			    SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
			  )
		)`, job.EndDate, job.StartDate)
	}

	query = query.Order("serialnumber ASC").Limit(quantity)

	err := query.Find(&devices).Error
	return devices, err
}
