package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type ScannerHandler struct {
	deviceRepo        *repository.DeviceRepository
	jobRepo           *repository.JobRepository
	customerRepo      *repository.CustomerRepository
	caseRepo          *repository.CaseRepository
	rentalEquipmentRepo *repository.RentalEquipmentRepository
}

func NewScannerHandler(jobRepo *repository.JobRepository, deviceRepo *repository.DeviceRepository, customerRepo *repository.CustomerRepository, caseRepo *repository.CaseRepository, rentalEquipmentRepo *repository.RentalEquipmentRepository) *ScannerHandler {
	return &ScannerHandler{
		deviceRepo:        deviceRepo,
		jobRepo:           jobRepo,
		customerRepo:      customerRepo,
		caseRepo:          caseRepo,
		rentalEquipmentRepo: rentalEquipmentRepo,
	}
}

func (h *ScannerHandler) ScanJobSelection(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	// Free devices from completed jobs first
	err := h.jobRepo.FreeDevicesFromCompletedJobs()
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to free devices from completed jobs: %v\n", err)
	}
	
	// Get all jobs first
	allJobs, err := h.jobRepo.List(&models.FilterParams{})
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}
	
	// Filter out paid jobs - only show open and in progress jobs
	var jobs []models.JobWithDetails
	for _, job := range allJobs {
		if job.StatusName != "paid" {
			jobs = append(jobs, job)
		}
	}

	// Get available device count for today
	today := time.Now()
	availableCount, err := h.deviceRepo.CountAvailableDevicesForDate(today)
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	// Get total device count for calculation
	totalDeviceCount, err := h.deviceRepo.GetTotalDeviceCount()
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}
	
	// Use the actual count of devices assigned to jobs, not the calculated difference
	assignedCount, _ := h.deviceRepo.CountDevicesAssignedToJobs(today)

	SafeHTML(c, http.StatusOK, "scan_select_job.html", gin.H{
		"title":           "Select Job for Scanning",
		"jobs":            jobs,
		"totalDevices":    availableCount,
		"assignedDevices": assignedCount,
		"totalDeviceCount": totalDeviceCount,
		"user":            user,
	})
}

func (h *ScannerHandler) ScanJob(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	// Check if this is a mobile device and redirect to optimized mobile scanner
	userAgent := c.GetHeader("User-Agent")
	isMobile := h.isMobileDevice(userAgent)

	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/error?code=400&message=Bad Request&details=Invalid job ID")
		return
	}

	// For mobile devices, redirect to optimized mobile scanner
	if isMobile {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/mobile/scanner/%d", jobID))
		return
	}

	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/error?code=404&message=Job Not Found&details=Job not found")
		return
	}

	// Debug logging for customer
	fmt.Printf("🔧 DEBUG ScanJob: Job %d has CustomerID: %d\n", jobID, job.CustomerID)
	fmt.Printf("🔧 DEBUG ScanJob: Customer loaded - ID: %d, Company: %v, FirstName: %v, LastName: %v\n", 
		job.Customer.CustomerID, job.Customer.CompanyName, job.Customer.FirstName, job.Customer.LastName)
	fmt.Printf("🔧 DEBUG ScanJob: GetDisplayName returns: '%s'\n", job.Customer.GetDisplayName())
	
	// Try to manually load customer if the preloaded one is empty
	if job.Customer.CustomerID == 0 && job.CustomerID > 0 {
		fmt.Printf("🔧 DEBUG ScanJob: Customer not preloaded, trying manual load for CustomerID: %d\n", job.CustomerID)
		customer, err := h.customerRepo.GetByID(job.CustomerID)
		if err != nil {
			fmt.Printf("🔧 DEBUG ScanJob: Failed to manually load customer: %v\n", err)
		} else {
			fmt.Printf("🔧 DEBUG ScanJob: Manually loaded customer - ID: %d, Company: %v, FirstName: %v, LastName: %v\n", 
				customer.CustomerID, customer.CompanyName, customer.FirstName, customer.LastName)
			job.Customer = *customer
		}
	}

	// Get ONLY total device count - no other queries for maximum speed
	totalDevices, err := h.jobRepo.GetJobDeviceCount(uint(jobID))
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	// ULTRA-FAST MODE: Load absolutely nothing upfront
	// All data will be loaded on-demand via AJAX
	productGroups := make(map[string]*ProductGroup)

	// Get available cases for case scanning functionality
	cases, err := h.caseRepo.List(&models.FilterParams{})
	if err != nil {
		// If we can't get cases, continue without them - don't fail the page
		cases = []models.Case{}
	}

	// Get available rental equipment
	var rentalEquipment []models.RentalEquipment
	err = h.rentalEquipmentRepo.GetAllRentalEquipment(&rentalEquipment)
	if err != nil {
		// If we can't get rental equipment, continue without them - don't fail the page
		rentalEquipment = []models.RentalEquipment{}
	}

	// Get existing job rental equipment
	var jobRentalEquipment []models.JobRentalEquipment
	err = h.rentalEquipmentRepo.GetJobRentalEquipment(uint(jobID), &jobRentalEquipment)
	if err != nil {
		// If we can't get job rental equipment, continue without them - don't fail the page
		jobRentalEquipment = []models.JobRentalEquipment{}
	}

	c.HTML(http.StatusOK, "scan_job.html", gin.H{
		"title":              "Scanning Job #" + strconv.FormatUint(jobID, 10),
		"job":                job,
		"productGroups":      productGroups,
		"totalDevices":       totalDevices,
		"DeviceCount":        totalDevices,  // Add DeviceCount for template compatibility
		"cases":              cases,
		"rentalEquipment":    rentalEquipment,
		"jobRentalEquipment": jobRentalEquipment,
		"user":               user,
	})
}

type ScanDeviceRequest struct {
	JobID    uint     `json:"job_id" binding:"required"`
	DeviceID string   `json:"device_id" binding:"required"`
	Price    *float64 `json:"price"`
}

type ScanCaseRequest struct {
	JobID  uint `json:"job_id" binding:"required"`
	CaseID uint `json:"case_id" binding:"required"`
}

func (h *ScannerHandler) ScanDevice(c *gin.Context) {
	fmt.Printf("🚨 DEBUG SCANNER: ScanDevice called!\n")
	
	var req ScanDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ DEBUG SCANNER: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🚨 DEBUG SCANNER: Request - JobID: %d, DeviceID: %s\n", req.JobID, req.DeviceID)

	// Try to get device by ID first, then by serial number
	var device *models.Device
	var err error

	device, err = h.deviceRepo.GetByID(req.DeviceID)
	if err != nil {
		// Try by serial number
		device, err = h.deviceRepo.GetBySerialNo(req.DeviceID)
		if err != nil {
			fmt.Printf("❌ DEBUG SCANNER: Device not found: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
	}

	fmt.Printf("✅ DEBUG SCANNER: Device found: %s\n", device.DeviceID)

	// Get job details to check date range
	job, err := h.jobRepo.GetByID(req.JobID)
	if err != nil {
		fmt.Printf("❌ DEBUG SCANNER: Job not found: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	fmt.Printf("🚨 DEBUG SCANNER: Job %d dates: %v to %v\n", req.JobID, job.StartDate, job.EndDate)

	// Check if device is available for this job's date range
	fmt.Printf("🔍 DEBUG SCANNER: Checking availability for device %s, job %d, dates: %v to %v\n",
		device.DeviceID, req.JobID, job.StartDate, job.EndDate)

	isAvailable, conflictingAssignment, err := h.deviceRepo.IsDeviceAvailableForJob(device.DeviceID, req.JobID, job.StartDate, job.EndDate)
	if err != nil {
		fmt.Printf("❌ DEBUG SCANNER: Availability check error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check device availability",
			"details": err.Error(),
			"device_id": device.DeviceID,
		})
		return
	}

	fmt.Printf("🚨 DEBUG SCANNER: Device available: %t\n", isAvailable)

	if !isAvailable {
		if conflictingAssignment != nil {
			// Get conflicting job details for error message
			conflictingJob, _ := h.jobRepo.GetByID(conflictingAssignment.JobID)
			if conflictingJob != nil && conflictingAssignment.JobID == req.JobID {
				c.JSON(http.StatusConflict, gin.H{
					"error": fmt.Sprintf("Device is already assigned to this job #%d", req.JobID),
				})
			} else if conflictingJob != nil {
				c.JSON(http.StatusConflict, gin.H{
					"error": fmt.Sprintf("Device is already assigned to job #%d from %s to %s", 
						conflictingAssignment.JobID,
						conflictingJob.StartDate.Format("2006-01-02"),
						conflictingJob.EndDate.Format("2006-01-02")),
				})
			} else {
				c.JSON(http.StatusConflict, gin.H{
					"error": fmt.Sprintf("Device is already assigned to job #%d", conflictingAssignment.JobID),
				})
			}
		} else {
			c.JSON(http.StatusConflict, gin.H{
				"error":  "Device is not available",
				"device": device,
			})
		}
		return
	}

	// Assign device to job
	var price float64
	// Only use custom price if explicitly provided, otherwise pass 0 (which means NULL in DB)
	if req.Price != nil {
		price = *req.Price
	} else {
		price = 0.0 // This will result in NULL custom_price in database
	}

	if err := h.jobRepo.AssignDevice(req.JobID, device.DeviceID, price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device successfully assigned to job",
		"device":  device,
		"price":   price,
	})
}

func (h *ScannerHandler) RemoveDevice(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	if err := h.jobRepo.RemoveDevice(uint(jobID), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device removed from job successfully"})
}

type BulkRemoveRequest struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
}

func (h *ScannerHandler) BulkRemoveDevices(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req BulkRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.DeviceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No device IDs provided"})
		return
	}

	var successCount, errorCount int
	var errors []string

	for _, deviceID := range req.DeviceIDs {
		if err := h.jobRepo.RemoveDevice(uint(jobID), deviceID); err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("Failed to remove %s: %s", deviceID, err.Error()))
		} else {
			successCount++
		}
	}

	result := gin.H{
		"message":       fmt.Sprintf("Bulk removal completed: %d succeeded, %d failed", successCount, errorCount),
		"success_count": successCount,
		"error_count":   errorCount,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	if errorCount > 0 && successCount == 0 {
		c.JSON(http.StatusInternalServerError, result)
	} else {
		c.JSON(http.StatusOK, result)
	}
}

func (h *ScannerHandler) ScanCase(c *gin.Context) {
	var req ScanCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the case and its devices
	case_, err := h.caseRepo.GetByID(req.CaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	// Get all devices in the case
	devicesInCase, err := h.caseRepo.GetDevicesInCase(req.CaseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get devices in case"})
		return
	}

	if len(devicesInCase) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Case is empty - no devices to assign"})
		return
	}

	// Track results
	var results []map[string]interface{}
	successCount := 0
	errorCount := 0

	// Assign all devices in the case to the job
	for _, deviceCase := range devicesInCase {
		device := deviceCase.Device
		
		// Check if device is available
		if device.Status != "free" {
			results = append(results, map[string]interface{}{
				"device_id": device.DeviceID,
				"success":   false,
				"message":   "Device is not available (status: " + device.Status + ")",
			})
			errorCount++
			continue
		}

		// Assign device to job using default pricing (no custom price for case scanning)
		if err := h.jobRepo.AssignDevice(req.JobID, device.DeviceID, 0.0); err != nil {
			results = append(results, map[string]interface{}{
				"device_id": device.DeviceID,
				"success":   false,
				"message":   err.Error(),
			})
			errorCount++
		} else {
			results = append(results, map[string]interface{}{
				"device_id": device.DeviceID,
				"success":   true,
				"message":   "Device assigned successfully",
			})
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("Case scan complete: %d devices assigned, %d errors", successCount, errorCount),
		"case_id":       req.CaseID,
		"case_name":     case_.Name,
		"total_devices": len(devicesInCase),
		"success_count": successCount,
		"error_count":   errorCount,
		"results":       results,
	})
}

// AddRentalToJob adds rental equipment to a job from the scan page
func (h *ScannerHandler) AddRentalToJob(c *gin.Context) {
	var request models.AddRentalToJobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get rental price from equipment
	var equipment models.RentalEquipment
	err := h.rentalEquipmentRepo.GetRentalEquipmentByID(request.EquipmentID, &equipment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rental equipment not found"})
		return
	}

	// Calculate total cost
	totalCost := equipment.RentalPrice * float64(request.Quantity) * float64(request.DaysUsed)

	jobRental := &models.JobRentalEquipment{
		JobID:       request.JobID,
		EquipmentID: request.EquipmentID,
		Quantity:    request.Quantity,
		DaysUsed:    request.DaysUsed,
		TotalCost:   totalCost,
		Notes:       request.Notes,
	}

	err = h.rentalEquipmentRepo.AddRentalToJob(jobRental)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add rental to job: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Rental equipment added to job successfully",
		"jobRental":  jobRental,
		"totalCost":  totalCost,
		"equipment":  equipment,
	})
}

// RemoveRentalFromJob removes rental equipment from a job
func (h *ScannerHandler) RemoveRentalFromJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	equipmentID, err := strconv.ParseUint(c.Param("equipmentId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment ID"})
		return
	}

	err = h.rentalEquipmentRepo.RemoveRentalFromJob(uint(jobID), uint(equipmentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to remove rental from job: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rental equipment removed from job successfully"})
}

// GetJobDevicesAJAX handles AJAX requests for loading devices by product (performance optimized)
func (h *ScannerHandler) GetJobDevicesAJAX(c *gin.Context) {
	jobIDStr := c.Param("jobid")
	productName := c.Query("product")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "50")

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	// Get paginated devices for the product
	devices, err := h.jobRepo.GetJobDevicesPaginated(uint(jobID), productName, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"page":    page,
		"limit":   limit,
		"count":   len(devices),
	})
}

// GetJobDeviceGroupsAJAX handles AJAX requests for loading device groups (ultra-fast)
func (h *ScannerHandler) GetJobDeviceGroupsAJAX(c *gin.Context) {
	jobIDStr := c.Param("jobid")

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get product summaries only (super fast query)
	productSummaries, err := h.jobRepo.GetJobDeviceProductSummary(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load device groups"})
		return
	}

	// Convert to lightweight format for JSON
	groups := make([]map[string]interface{}, 0)
	for _, summary := range productSummaries {
		group := map[string]interface{}{
			"productName": summary.ProductName,
			"count":       summary.Count,
			"productID":   "",
			"pricePerDay": 0.0,
		}

		if summary.Product != nil {
			group["productID"] = summary.Product.ProductID
			if summary.Product.ItemCostPerDay != nil {
				group["pricePerDay"] = *summary.Product.ItemCostPerDay
			}
		}

		groups = append(groups, group)
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"total":  len(groups),
	})
}

// isMobileDevice checks if the user agent indicates a mobile device
func (h *ScannerHandler) isMobileDevice(userAgent string) bool {
	userAgent = strings.ToLower(userAgent)
	mobileKeywords := []string{
		"mobile", "iphone", "ipod", "android", "blackberry",
		"nokia", "opera mini", "windows mobile", "windows phone",
		"iemobile", "wpdesktop", "palm", "phone", "tablet",
		"kindle", "silk", "fennec", "maemo", "iris", "avant",
		"nook", "chromeos", "crkey", "mobile", "samsung",
		"htc", "lg", "mot", "sony", "ericsson", "alcatel",
		"asus", "bird", "dell", "docomo", "huawei", "kyocera",
		"lenovo", "lge", "panasonic", "philips", "samsung",
		"sharp", "toshiba", "vodafone", "xiaomi", "oppo",
		"vivo", "oneplus", "pixel",
	}

	for _, keyword := range mobileKeywords {
		if strings.Contains(userAgent, keyword) {
			return true
		}
	}

	return false
}