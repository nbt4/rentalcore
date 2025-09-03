package repository

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"
	"go-barcode-webapp/internal/models"
)

type DeviceRepository struct {
	db *Database
}

func NewDeviceRepository(db *Database) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// GetDB returns the underlying database connection for advanced queries
func (r *DeviceRepository) GetDB() *Database {
	return r.db
}

func (r *DeviceRepository) Create(device *models.Device) error {
	log.Printf("🚨 DEVICE CREATION: Creating device %s with productID %v", device.DeviceID, device.ProductID)
	log.Printf("🚨 DEVICE CREATION: Stack trace: %s", string(debug.Stack()))
	
	// Check if this is being called during package operations
	stackTrace := string(debug.Stack())
	if strings.Contains(stackTrace, "equipment_package") || strings.Contains(stackTrace, "UpdateDeviceAssociations") || strings.Contains(stackTrace, "package") {
		log.Printf("❌ DEVICE CREATION: Blocked device creation during package operations")
		return fmt.Errorf("device creation blocked during package operations - device %s does not exist", device.DeviceID)
	}
	
	// Generate device ID if not provided
	if device.DeviceID == "" {
		generatedID, err := r.generateDeviceID(device)
		if err != nil {
			log.Printf("❌ DEVICE CREATION: Failed to generate device ID: %v", err)
			return fmt.Errorf("failed to generate device ID: %v", err)
		}
		device.DeviceID = generatedID
		log.Printf("✅ DEVICE CREATION: Generated device ID: %s", device.DeviceID)
	}
	
	return r.db.Create(device).Error
}

func (r *DeviceRepository) GetByID(deviceID string) (*models.Device, error) {
	var device models.Device
	err := r.db.Where("deviceID = ?", deviceID).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Preload("Product.Brand").
		Preload("Product.Manufacturer").
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) GetBySerialNo(serialNo string) (*models.Device, error) {
	var device models.Device
	err := r.db.Where("serialnumber = ?", serialNo).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Preload("Product.Brand").
		Preload("Product.Manufacturer").
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) Update(device *models.Device) error {
	return r.db.Save(device).Error
}

func (r *DeviceRepository) Delete(deviceID string) error {
	log.Printf("🗑️ DEVICE DELETION: Deleting device %s", deviceID)
	err := r.db.Where("deviceID = ?", deviceID).Delete(&models.Device{}).Error
	if err != nil {
		log.Printf("❌ DEVICE DELETION: Failed to delete device %s: %v", deviceID, err)
	} else {
		log.Printf("✅ DEVICE DELETION: Successfully deleted device %s", deviceID)
	}
	return err
}

func (r *DeviceRepository) List(params *models.FilterParams) ([]models.DeviceWithJobInfo, error) {
	startTime := time.Now()
	log.Printf("⏱️  DeviceRepository.List() started")

	var devices []models.Device

	// Set default pagination if not provided
	limit := params.Limit
	if limit <= 0 {
		limit = 20 // Default devices per page
	}
	
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	// Simple query without complex joins for better performance
	query := r.db.Model(&models.Device{})
	
	// Always preload Product with Category for proper display
	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Preload("Product").Preload("Product.Category").
			Joins("LEFT JOIN products ON products.productID = devices.productID").
			Where("devices.deviceID LIKE ? OR devices.serialnumber LIKE ? OR products.name LIKE ?", searchPattern, searchPattern, searchPattern)
	} else {
		// For normal list view, preload Product with Category
		query = query.Preload("Product").Preload("Product.Category")
	}

	query = query.Limit(limit).Offset(offset).Order("deviceID DESC")

	queryStart := time.Now()
	err := query.Find(&devices).Error
	queryTime := time.Since(queryStart)
	log.Printf("⏱️  Device query took: %v", queryTime)
	
	if err != nil {
		log.Printf("❌ Device query error: %v", err)
		return nil, err
	}
	
	// Skip job assignment check for better performance - we can add it back later if needed
	var result []models.DeviceWithJobInfo
	for _, device := range devices {
		result = append(result, models.DeviceWithJobInfo{
			Device:     device,
			JobID:      nil,
			IsAssigned: false,
		})
	}

	totalTime := time.Since(startTime)
	log.Printf("⏱️  DeviceRepository.List() completed in %v (found %d devices)", totalTime, len(result))

	return result, nil
}

func (r *DeviceRepository) ListWithCategories(params *models.FilterParams) ([]models.Device, error) {
	var devices []models.Device

	query := r.db.Model(&models.Device{}).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Brand").
		Preload("Product.Manufacturer")

	// Join products table for search and category filtering
	query = query.Joins("JOIN products ON products.productID = devices.productID")

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("devices.deviceID LIKE ? OR devices.serialnumber LIKE ? OR products.name LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Category filter
	if params.Category != "" {
		query = query.Joins("JOIN categories ON categories.categoryID = products.categoryID").
			Where("categories.name = ?", params.Category)
	}

	// Status filter
	if params.Status != "" {
		query = query.Where("devices.status = ?", params.Status)
	}

	// Filter devices not in any case (for case assignment)
	if params.AssignmentStatus == "not_in_case" {
		query = query.Where("devices.deviceID NOT IN (SELECT DISTINCT deviceID FROM devicescases)")
	}

	// Available filter (devices not in any case and with free status)
	if params.Available != nil && *params.Available {
		query = query.Where("devices.status = 'free' AND devices.deviceID NOT IN (SELECT DISTINCT deviceID FROM devicescases)")
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	query = query.Order("deviceID DESC")

	err := query.Find(&devices).Error
	return devices, err
}

func (r *DeviceRepository) GetByProductID(productID uint) ([]models.Device, error) {
	var devices []models.Device
	err := r.db.Where("productID = ?", productID).
		Preload("Product").
		Order("deviceID ASC").
		Find(&devices).Error
	return devices, err
}

func (r *DeviceRepository) GetAvailableDevices() ([]models.Device, error) {
	var devices []models.Device
	
	// Get devices that are available and not currently assigned to any active job (considering dates)
	currentDate := time.Now().Format("2006-01-02")
	err := r.db.Where(`status = 'free' AND deviceID NOT IN (
		SELECT DISTINCT jd.deviceID 
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID 
		WHERE j.startDate <= ? AND j.endDate >= ? AND j.statusID IN (
			SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
		)
	)`, currentDate, currentDate).Find(&devices).Error
	
	return devices, err
}

func (r *DeviceRepository) GetDevicesByCategory(category string) ([]models.Device, error) {
	var devices []models.Device
	err := r.db.Where("category = ? AND available = true", category).
		Find(&devices).Error
	return devices, err
}

func (r *DeviceRepository) CheckDeviceAvailability(deviceID uint) (bool, error) {
	var count int64
	err := r.db.Table("job_devices").
		Where("device_id = ? AND removed_at IS NULL", deviceID).
		Count(&count).Error
	
	return count == 0, err
}

func (r *DeviceRepository) GetDeviceJobHistory(deviceID uint) ([]models.JobDevice, error) {
	var jobDevices []models.JobDevice
	err := r.db.Where("device_id = ?", deviceID).
		Preload("Job").
		Preload("Job.Customer").
		Find(&jobDevices).Error
	
	return jobDevices, err
}

func (r *DeviceRepository) GetAvailableDevicesForCaseManagement() ([]models.Device, error) {
	var devices []models.Device
	
	// Get all devices with product information, regardless of status or case assignment
	err := r.db.Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Preload("Product.Brand").
		Preload("Product.Manufacturer").
		Find(&devices).Error
	
	return devices, err
}


func (r *DeviceRepository) GetDeviceStats(deviceID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get total number of jobs this device has been assigned to
	var totalJobs int64
	err := r.db.Model(&models.JobDevice{}).
		Where("deviceID = ?", deviceID).
		Count(&totalJobs).Error
	if err != nil {
		log.Printf("Error counting jobs for device %s: %v", deviceID, err)
		totalJobs = 0
	}
	
	// Get total earnings from jobs (simplified calculation)
	var totalEarnings float64
	err = r.db.Raw(`
		SELECT COALESCE(SUM(DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate) * COALESCE(p.itemcostperday, 0)), 0) as total_earnings
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID
		JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		WHERE jd.deviceID = ?
	`, deviceID).Scan(&totalEarnings).Error
	if err != nil {
		log.Printf("Error calculating earnings for device %s: %v", deviceID, err)
		totalEarnings = 0.0
	}
	
	// Get total days rented
	var totalDaysRented int64
	err = r.db.Raw(`
		SELECT COALESCE(SUM(DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate)), 0) as total_days
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID
		WHERE jd.deviceID = ?
	`, deviceID).Scan(&totalDaysRented).Error
	if err != nil {
		log.Printf("Error calculating days rented for device %s: %v", deviceID, err)
		totalDaysRented = 0
	}
	
	// Calculate average rental duration
	var averageRentalDuration float64
	if totalJobs > 0 {
		averageRentalDuration = float64(totalDaysRented) / float64(totalJobs)
	}
	
	// Get device product details for price per day
	var device models.Device
	err = r.db.Where("deviceID = ?", deviceID).Preload("Product").First(&device).Error
	if err != nil {
		log.Printf("Error getting device details for %s: %v", deviceID, err)
	}
	
	var pricePerDay float64
	var weight float64
	if device.Product != nil {
		if device.Product.ItemCostPerDay != nil {
			pricePerDay = *device.Product.ItemCostPerDay
		}
		if device.Product.Weight != nil {
			weight = *device.Product.Weight
		}
	}
	
	stats["totalJobs"] = totalJobs
	stats["totalEarnings"] = totalEarnings
	stats["totalDaysRented"] = totalDaysRented
	stats["averageRentalDuration"] = averageRentalDuration
	stats["pricePerDay"] = pricePerDay
	stats["weight"] = weight
	
	return stats, nil
}

// generateDeviceID generates a unique device ID based on the product category and existing devices
func (r *DeviceRepository) generateDeviceID(device *models.Device) (string, error) {
	// Default prefix if we can't determine from product
	prefix := "DEV"
	
	// If we have a product, try to determine a prefix based on product name
	if device.ProductID != nil {
		var product models.Product
		err := r.db.First(&product, *device.ProductID).Error
		if err == nil && product.Name != "" {
			prefix = r.generatePrefixFromProductName(product.Name)
		}
	}
	
	// Find the next available number for this prefix
	var maxNum int
	err := r.db.Raw(`
		SELECT COALESCE(MAX(CAST(SUBSTRING(deviceID, ?) AS UNSIGNED)), 0) as max_num 
		FROM devices 
		WHERE deviceID LIKE ?
	`, len(prefix)+1, prefix+"%").Scan(&maxNum).Error
	
	if err != nil {
		log.Printf("❌ Error finding max device number for prefix %s: %v", prefix, err)
		return "", fmt.Errorf("failed to find max device number: %v", err)
	}
	
	// Generate new device ID
	newNum := maxNum + 1
	deviceID := fmt.Sprintf("%s%04d", prefix, newNum)
	
	log.Printf("✅ Generated device ID: %s (prefix: %s, next number: %d)", deviceID, prefix, newNum)
	return deviceID, nil
}

// generatePrefixFromProductName creates a prefix based on the product name
func (r *DeviceRepository) generatePrefixFromProductName(productName string) string {
	// Simple mapping based on common patterns observed in existing data
	name := strings.ToLower(productName)
	
	// Audio/Lighting equipment
	if strings.Contains(name, "speaker") || strings.Contains(name, "stand") || strings.Contains(name, "lighting") {
		return "LFT"
	}
	
	// CO2 equipment
	if strings.Contains(name, "co2") || strings.Contains(name, "bottle") || strings.Contains(name, "hose") {
		return "CO2"
	}
	
	// Hazer/Fog equipment
	if strings.Contains(name, "hazer") || strings.Contains(name, "fog") || strings.Contains(name, "dmx") {
		return "FOG"
	}
	
	// Microphone/Audio equipment
	if strings.Contains(name, "microphone") || strings.Contains(name, "mic") || strings.Contains(name, "audio") {
		return "MHD"
	}
	
	// Accessories
	if strings.Contains(name, "accessory") || strings.Contains(name, "cable") || strings.Contains(name, "adapter") {
		return "ACC"
	}
	
	// External/Rental
	if strings.Contains(name, "external") || strings.Contains(name, "rental") || strings.Contains(name, "cleaning") {
		return "EXT"
	}
	
	// Default fallback
	return "DEV"
}

// GetAvailableDevicesForDate returns devices that are available on a specific date
// A device is available if it's not assigned to any job that overlaps with the given date
func (r *DeviceRepository) GetAvailableDevicesForDate(targetDate time.Time) ([]models.Device, error) {
	var devices []models.Device
	
	// Get all devices with 'free' status that are NOT assigned to jobs overlapping the target date
	// CORRECTED: Use >= for endDate comparison
	// This ensures devices are unavailable ON the end date and become available the day AFTER
	err := r.db.Where(`status = 'free' AND deviceID NOT IN (
		SELECT DISTINCT jd.deviceID 
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID 
		WHERE j.startDate <= ? AND j.endDate >= ? AND j.statusID IN (
			SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
		)
	)`, targetDate, targetDate).Find(&devices).Error
	
	return devices, err
}

// CountAvailableDevicesForDate returns the count of devices available on a specific date
func (r *DeviceRepository) CountAvailableDevicesForDate(targetDate time.Time) (int64, error) {
	var count int64
	
	// CORRECTED: Use >= for endDate comparison
	// This ensures devices are unavailable ON the end date and become available the day AFTER
	// Example: If endDate = 2025-07-19, devices are unavailable on 2025-07-19, available on 2025-07-20
	err := r.db.Model(&models.Device{}).Where(`status = 'free' AND deviceID NOT IN (
		SELECT DISTINCT jd.deviceID 
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID 
		WHERE j.startDate <= ? AND j.endDate >= ? AND j.statusID IN (
			SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
		)
	)`, targetDate, targetDate).Count(&count).Error
	
	return count, err
}

// GetTotalDeviceCount returns the total number of devices in the database
func (r *DeviceRepository) GetTotalDeviceCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Device{}).Count(&count).Error
	return count, err
}

// CountAssignedDevicesForDate returns the count of devices assigned to jobs on a specific date
func (r *DeviceRepository) CountAssignedDevicesForDate(targetDate time.Time) (int64, error) {
	var count int64
	
	err := r.db.Model(&models.Device{}).Where(`deviceID IN (
		SELECT DISTINCT jd.deviceID 
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID 
		WHERE j.startDate <= ? AND j.endDate >= ? AND j.statusID IN (
			SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
		)
	)`, targetDate, targetDate).Count(&count).Error
	
	return count, err
}

// CountDevicesByStatus returns the count of devices with a specific status
func (r *DeviceRepository) CountDevicesByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Device{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// CountDevicesAssignedToJobs returns the count of devices assigned to any job on a specific date
// This counts ALL devices in job assignments regardless of device status
func (r *DeviceRepository) CountDevicesAssignedToJobs(targetDate time.Time) (int64, error) {
	var count int64
	
	fmt.Printf("🔍 DEBUG: CountDevicesAssignedToJobs called with targetDate: %s\n", targetDate.Format("2006-01-02"))
	
	// CORRECTED: Use >= for endDate comparison
	// This ensures devices are unavailable ON the end date and become available the day AFTER
	err := r.db.Table("jobdevices jd").
		Joins("JOIN jobs j ON jd.jobID = j.jobID").
		Where("j.startDate <= ? AND j.endDate >= ? AND j.statusID IN (SELECT statusID FROM status WHERE status IN ('open', 'in_progress'))", targetDate, targetDate).
		Count(&count).Error
	
	fmt.Printf("🔍 DEBUG: Total devices assigned to jobs on %s: %d\n", targetDate.Format("2006-01-02"), count)
	
	return count, err
}

// IsDeviceAvailableForJob checks if a device is available for a specific job's date range
func (r *DeviceRepository) IsDeviceAvailableForJob(deviceID string, jobID uint, startDate, endDate *time.Time) (bool, *models.JobDevice, error) {
	// If no dates specified, use basic availability check
	if startDate == nil || endDate == nil {
		// Check if device exists and has 'free' status
		var device models.Device
		err := r.db.Where("deviceID = ? AND status = 'free'", deviceID).First(&device).Error
		if err != nil {
			return false, nil, err
		}
		
		// Check if already assigned to this specific job
		var existingAssignment models.JobDevice
		err = r.db.Where("deviceID = ? AND jobID = ?", deviceID, jobID).First(&existingAssignment).Error
		if err == nil {
			return false, &existingAssignment, nil // Already assigned to this job
		}
		
		// Check if assigned to any other job
		err = r.db.Where("deviceID = ?", deviceID).First(&existingAssignment).Error
		if err == nil {
			return false, &existingAssignment, nil // Assigned to another job
		}
		
		return true, nil, nil
	}
	
	// Check if device exists and has 'free' status
	var device models.Device
	err := r.db.Where("deviceID = ? AND status = 'free'", deviceID).First(&device).Error
	if err != nil {
		return false, nil, err
	}
	
	// Check for overlapping job assignments
	// CORRECTED: Use >= for endDate comparison
	// This ensures devices are unavailable ON the end date and become available the day AFTER
	var conflictingJob models.JobDevice
	err = r.db.Joins("JOIN jobs ON jobdevices.jobID = jobs.jobID").
		Where(`jobdevices.deviceID = ? 
			AND jobs.jobID != ? 
			AND jobs.startDate <= ? 
			AND jobs.endDate >= ? 
			AND jobs.statusID IN (
				SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
			)`, deviceID, jobID, endDate, startDate).
		First(&conflictingJob).Error
	
	if err == nil {
		// Found a conflicting assignment, get the job details
		var job models.Job
		r.db.Where("jobID = ?", conflictingJob.JobID).First(&job)
		conflictingJob.Job = job
		return false, &conflictingJob, nil
	}
	
	return true, nil, nil
}

// GetTotalCount returns the total number of devices
func (r *DeviceRepository) GetTotalCount() (int, error) {
	var count int64
	err := r.db.Model(&models.Device{}).Count(&count).Error
	return int(count), err
}

// GetAvailableDevicesForJob returns devices available for a specific job's date range
func (r *DeviceRepository) GetAvailableDevicesForJob(jobID uint, startDate, endDate *time.Time) ([]models.Device, error) {
	var devices []models.Device
	
	// If no dates provided, use the basic availability check
	if startDate == nil || endDate == nil {
		return r.GetAvailableDevices()
	}
	
	// Get devices that are not assigned to overlapping jobs
	err := r.db.Where(`status = 'free' AND deviceID NOT IN (
		SELECT DISTINCT jd.deviceID 
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID 
		WHERE j.jobID != ? 
			AND j.startDate <= ? 
			AND j.endDate >= ? 
			AND j.statusID IN (
				SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
			)
	)`, jobID, endDate, startDate).Find(&devices).Error
	
	return devices, err
}

// IsDeviceCurrentlyAssigned checks if a device is currently assigned to an active job
// considering job dates and status. Returns true if the device should show as "assigned"
func (r *DeviceRepository) IsDeviceCurrentlyAssigned(deviceID string) (bool, *uint, error) {
	currentDate := time.Now().Format("2006-01-02")
	
	var assignment models.JobDevice
	err := r.db.Joins("JOIN jobs ON jobdevices.jobID = jobs.jobID").
		Where(`jobdevices.deviceID = ? 
			AND jobs.startDate <= ? 
			AND jobs.endDate >= ? 
			AND jobs.statusID IN (
				SELECT statusID FROM status WHERE status IN ('open', 'in_progress')
			)`, deviceID, currentDate, currentDate).
		First(&assignment).Error
	
	if err != nil {
		if err.Error() == "record not found" {
			return false, nil, nil // Not assigned
		}
		return false, nil, err // Database error
	}
	
	return true, &assignment.JobID, nil
}