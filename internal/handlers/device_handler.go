package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
	"sync"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
)

// Simple cache for devices
type DeviceCache struct {
	data      []models.DeviceWithJobInfo
	timestamp time.Time
	mutex     sync.RWMutex
}

// Tree cache for optimized tree data
type TreeCache struct {
	data      []TreeCategory
	timestamp time.Time
	mutex     sync.RWMutex
}

var deviceCache = &DeviceCache{
	timestamp: time.Time{}, // Force cache miss initially - CLEARED FOR CATEGORY RELATIONSHIP FIX
}

var treeCache = &TreeCache{
	timestamp: time.Time{}, // Force cache miss initially - CLEARED FOR HIERARCHY FIX
}

type DeviceHandler struct {
	deviceRepo     *repository.DeviceRepository
	barcodeService *services.BarcodeService
	productRepo    *repository.ProductRepository
}

func NewDeviceHandler(deviceRepo *repository.DeviceRepository, barcodeService *services.BarcodeService, productRepo *repository.ProductRepository) *DeviceHandler {
	return &DeviceHandler{
		deviceRepo:     deviceRepo,
		barcodeService: barcodeService,
		productRepo:    productRepo,
	}
}


// Web interface handlers
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	
	user, _ := GetCurrentUser(c)
	
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=400&message=Bad Request&details=%s", err.Error()))
		return
	}
	
	// FIX: Ensure search parameter is properly handled
	searchParam := c.Query("search")
	if searchParam != "" {
		params.SearchTerm = searchParam
	}

	// Handle pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	
	limit := 20 // Devices per page
	params.Limit = limit
	params.Offset = (page - 1) * limit
	params.Page = page

	viewType := c.DefaultQuery("view", "list") // Default to list view

	// Use cache for basic list view without search (but not for tree or categorized views)
	var devices []models.DeviceWithJobInfo
	var err error
	
	if params.SearchTerm == "" && page == 1 && viewType == "list" {
		// Try to use cache for first page without search
		deviceCache.mutex.RLock()
		if time.Since(deviceCache.timestamp) < 30*time.Second && len(deviceCache.data) > 0 {
			// Use cached data
			devices = deviceCache.data
			if len(devices) > limit {
				devices = devices[:limit]
			}
			deviceCache.mutex.RUnlock()
		} else {
			deviceCache.mutex.RUnlock()
			
			// Fetch fresh data using ListWithCategories to ensure categories are loaded
			deviceList, err := h.deviceRepo.ListWithCategories(params)
				
			// Convert to DeviceWithJobInfo format with proper assignment checking
			devices = make([]models.DeviceWithJobInfo, len(deviceList))
			for i, device := range deviceList {
				// Check if device is currently assigned to an active job
				isAssigned, jobID, err := h.deviceRepo.IsDeviceCurrentlyAssigned(device.DeviceID)
				if err != nil {
						isAssigned = false
					jobID = nil
				}
				
				devices[i] = models.DeviceWithJobInfo{
					Device:     device,
					JobID:      jobID,
					IsAssigned: isAssigned,
				}
			}
			
			if err != nil {
					c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
				return
			}
			
			// Cache the result
			deviceCache.mutex.Lock()
			deviceCache.data = devices
			deviceCache.timestamp = time.Now()
			deviceCache.mutex.Unlock()
		}
	} else {
		// For search or pagination, use ListWithCategories to ensure categories are loaded
		deviceList, err := h.deviceRepo.ListWithCategories(params)
		
		// Convert to DeviceWithJobInfo format with proper assignment checking
		devices = make([]models.DeviceWithJobInfo, len(deviceList))
		for i, device := range deviceList {
			// Check if device is currently assigned to an active job
			isAssigned, jobID, err := h.deviceRepo.IsDeviceCurrentlyAssigned(device.DeviceID)
			if err != nil {
				isAssigned = false
				jobID = nil
			}
			
			devices[i] = models.DeviceWithJobInfo{
				Device:     device,
				JobID:      jobID,
				IsAssigned: isAssigned,
			}
		}
		
		if err != nil {
			c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
			return
		}
	}
	
	// Calculate pagination info for all list view requests (both cached and fresh)
	var totalDevices int
	var totalPages int
	if viewType == "list" {
		// Get total device count for pagination
		totalDevices, err = h.deviceRepo.GetTotalCount()
		if err != nil {
			totalDevices = 0
		}
		
		totalPages = (totalDevices + limit - 1) / limit // Ceiling division
		if totalPages == 0 {
			totalPages = 1
		}
	}
	if viewType == "tree" {
		// For tree view, load tree data and render in the main template
		treeData, err := h.buildTreeData()
		if err != nil {
			// Fall back to list view instead of error page
			SafeHTML(c, http.StatusOK, "devices_standalone.html", gin.H{
				"title":         "Devices (Tree Error - Showing List)",
				"devices":       devices,
				"params":        params,
				"user":          user,
				"viewType":      "list", // Force list view
				"currentPage":   "devices",
				"treeError":     err.Error(),
			})
			return
		}
		
		if len(treeData) == 0 {
			SafeHTML(c, http.StatusOK, "devices_standalone.html", gin.H{
				"title":         "Devices (Empty Tree - Showing List)",
				"devices":       devices,
				"params":        params,
				"user":          user,
				"viewType":      "list", // Force list view
				"currentPage":   "devices",
				"treeError":     "No categories found for tree view",
			})
			return
		}
		
		SafeHTML(c, http.StatusOK, "devices_standalone.html", gin.H{
			"title":       "Device Tree View",
			"params":      params,
			"user":        user,
			"viewType":    "tree",
			"currentPage": "devices",
			"treeData":    treeData,
		})
	} else {
		// Safe template rendering with error handling
		SafeHTML(c, http.StatusOK, "devices_standalone.html", gin.H{
			"title":         "Devices",
			"devices":       devices,
			"params":        params,
			"user":          user,
			"viewType":      "list",
			"categorized":   false,
			"currentPage":   "devices", // For navbar highlighting
			"pageNumber":    page,      // For pagination
			"hasNextPage":   page < totalPages,
			"totalPages":    totalPages,
			"totalDevices":  totalDevices,
		})
	}
}

func (h *DeviceHandler) NewDeviceForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	products, err := h.productRepo.List(&models.FilterParams{})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.HTML(http.StatusOK, "device_form.html", gin.H{
		"title":    "New Device",
		"device":   &models.Device{},
		"products": products,
		"user":     user,
	})
}

func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	
	// Get form values
	serialNumber := c.PostForm("serialnumber")
	status := c.PostForm("status")
	notes := c.PostForm("notes")
	quantityStr := c.PostForm("quantity")
	
	
	if status == "" {
		status = "free"
	}
	
	// Parse quantity (default to 1 if not provided or invalid)
	quantity := 1
	if quantityStr != "" {
		if q, err := strconv.Atoi(quantityStr); err == nil && q > 0 && q <= 100 {
			quantity = q
		}
	}
	
	var productID *uint
	if productIDStr := c.PostForm("productID"); productIDStr != "" {
		if pid, err := strconv.ParseUint(productIDStr, 10, 32); err == nil {
			prodID := uint(pid)
			productID = &prodID
		}
	}
	
	if productID == nil {
		user, _ := GetCurrentUser(c)
		products, _ := h.productRepo.List(&models.FilterParams{})
		c.HTML(http.StatusBadRequest, "device_form.html", gin.H{
			"title":    "New Device",
			"device":   &models.Device{},
			"products": products,
			"error":    "Please select a product",
			"user":     user,
		})
		return
	}
	
	
	// Create multiple devices
	createdDevices := make([]models.Device, 0, quantity)
	var lastError error
	
	for i := 0; i < quantity; i++ {
		device := models.Device{
			DeviceID:     "", // Let database generate the ID automatically
			ProductID:    productID,
			Status:       status,
		}
		
		// Handle optional string fields
		// For serial numbers, append index if creating multiple devices
		if serialNumber != "" {
			if quantity > 1 {
				indexedSerial := fmt.Sprintf("%s-%02d", serialNumber, i+1)
				device.SerialNumber = &indexedSerial
			} else {
				device.SerialNumber = &serialNumber
			}
		}
		
		if notes != "" {
			device.Notes = &notes
		}
		
		// Handle date fields
		if purchaseDateStr := c.PostForm("purchase_date"); purchaseDateStr != "" {
			if purchaseDate, err := time.Parse("2006-01-02", purchaseDateStr); err == nil {
				device.PurchaseDate = &purchaseDate
			}
		}
		if lastMaintenanceStr := c.PostForm("last_maintenance"); lastMaintenanceStr != "" {
			if lastMaintenance, err := time.Parse("2006-01-02", lastMaintenanceStr); err == nil {
				device.LastMaintenance = &lastMaintenance
			}
		}

		
		if err := h.deviceRepo.Create(&device); err != nil {
			lastError = err
			break
		}
		
		createdDevices = append(createdDevices, device)
	}
	
	// Handle errors
	if lastError != nil {
		user, _ := GetCurrentUser(c)
		products, _ := h.productRepo.List(&models.FilterParams{})
		errorMsg := fmt.Sprintf("Error creating device %d of %d: %v", len(createdDevices)+1, quantity, lastError)
		if len(createdDevices) > 0 {
			errorMsg += fmt.Sprintf(" (%d devices were created successfully before the error)", len(createdDevices))
		}
		c.HTML(http.StatusInternalServerError, "device_form.html", gin.H{
			"title":    "New Device",
			"device":   &models.Device{},
			"products": products,
			"error":    errorMsg,
			"user":     user,
		})
		return
	}

	c.Redirect(http.StatusFound, "/devices")
}

func (h *DeviceHandler) GetDevice(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	deviceID := c.Param("id")

	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Device not found", "user": user})
		return
	}

	c.HTML(http.StatusOK, "device_detail.html", gin.H{
		"device": device,
		"user":   user,
	})
}

func (h *DeviceHandler) EditDeviceForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	deviceID := c.Param("id")

	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Device not found", "user": user})
		return
	}

	products, err := h.productRepo.List(&models.FilterParams{})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.HTML(http.StatusOK, "device_form.html", gin.H{
		"title":    "Edit Device",
		"device":   device,
		"products": products,
		"user":     user,
	})
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	deviceID := c.Param("id")
	serialNumber := c.PostForm("serialnumber")
	status := c.PostForm("status")
	notes := c.PostForm("notes")
	
	var productID *uint
	if productIDStr := c.PostForm("productID"); productIDStr != "" {
		if pid, err := strconv.ParseUint(productIDStr, 10, 32); err == nil {
			prodID := uint(pid)
			productID = &prodID
		}
	}
	
	device := models.Device{
		DeviceID:  deviceID,
		ProductID: productID,
		Status:    status,
	}
	
	// Handle optional string fields
	if serialNumber != "" {
		device.SerialNumber = &serialNumber
	}
	if notes != "" {
		device.Notes = &notes
	}
	
	// Handle date fields
	if purchaseDateStr := c.PostForm("purchase_date"); purchaseDateStr != "" {
		if purchaseDate, err := time.Parse("2006-01-02", purchaseDateStr); err == nil {
			device.PurchaseDate = &purchaseDate
		}
	}
	if lastMaintenanceStr := c.PostForm("last_maintenance"); lastMaintenanceStr != "" {
		if lastMaintenance, err := time.Parse("2006-01-02", lastMaintenanceStr); err == nil {
			device.LastMaintenance = &lastMaintenance
		}
	}

	if err := h.deviceRepo.Update(&device); err != nil {
		user, _ := GetCurrentUser(c)
		products, _ := h.productRepo.List(&models.FilterParams{})
		c.HTML(http.StatusInternalServerError, "device_form.html", gin.H{
			"title":    "Edit Device",
			"device":   &device,
			"products": products,
			"error":    err.Error(),
			"user":     user,
		})
		return
	}

	c.Redirect(http.StatusFound, "/devices")
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")

	if err := h.deviceRepo.Delete(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

func (h *DeviceHandler) GetDeviceQR(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	deviceID := c.Param("id")

	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Device not found", "user": user})
		return
	}

	// Use serial number if available, otherwise use device ID
	identifier := deviceID
	if device.SerialNumber != nil && *device.SerialNumber != "" {
		identifier = *device.SerialNumber
	}
	
	qrCode, err := h.barcodeService.GenerateQRCode(identifier, 256)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.Data(http.StatusOK, "image/png", qrCode)
}

func (h *DeviceHandler) GetDeviceBarcode(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	deviceID := c.Param("id")

	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Device not found", "user": user})
		return
	}

	// Use serial number if available, otherwise use device ID
	identifier := deviceID
	if device.SerialNumber != nil && *device.SerialNumber != "" {
		identifier = *device.SerialNumber
	}
	
	barcode, err := h.barcodeService.GenerateBarcode(identifier)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.Data(http.StatusOK, "image/png", barcode)
}

func (h *DeviceHandler) GetAvailableDevices(c *gin.Context) {
	devices, err := h.deviceRepo.GetAvailableDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}

// API handlers for tree view
func (h *DeviceHandler) GetDevicesByCategory(c *gin.Context) {
	categoryID := c.Param("id")
	
	categoryIDUint, err := strconv.ParseUint(categoryID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}
	
	devices, err := h.productRepo.GetDevicesByCategory(uint(categoryIDUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, devices)
}

func (h *DeviceHandler) GetDevicesBySubcategory(c *gin.Context) {
	subcategoryID := c.Param("id")
	
	devices, err := h.productRepo.GetDevicesBySubcategory(subcategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, devices)
}

func (h *DeviceHandler) GetDevicesBySubbiercategory(c *gin.Context) {
	subbiercategoryID := c.Param("id")
	
	devices, err := h.productRepo.GetDevicesBySubbiercategory(subbiercategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, devices)
}

// API handlers
func (h *DeviceHandler) ListDevicesAPI(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use the new method with categories for case management
	devices, err := h.deviceRepo.ListWithCategories(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}

func (h *DeviceHandler) CreateDeviceAPI(c *gin.Context) {
	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.deviceRepo.Create(&device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, device)
}

func (h *DeviceHandler) GetDeviceAPI(c *gin.Context) {
	deviceID := c.Param("id")
	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		// Try by serial number if not found by ID
		device, err = h.deviceRepo.GetBySerialNo(deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"device": device})
}

func (h *DeviceHandler) UpdateDeviceAPI(c *gin.Context) {
	deviceID := c.Param("id")

	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device.DeviceID = deviceID
	if err := h.deviceRepo.Update(&device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

func (h *DeviceHandler) DeleteDeviceAPI(c *gin.Context) {
	deviceID := c.Param("id")

	if err := h.deviceRepo.Delete(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

func (h *DeviceHandler) GetDeviceStatsAPI(c *gin.Context) {
	deviceID := c.Param("id")
	
	// Get device details
	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Get device statistics
	stats, err := h.deviceRepo.GetDeviceStats(deviceID)
	if err != nil {
		// Return basic device info even if stats fail
		c.JSON(http.StatusOK, gin.H{
			"device": device,
			"stats": gin.H{
				"totalJobs": 0,
				"totalEarnings": 0.0,
				"totalDaysRented": 0,
				"averageRentalDuration": 0.0,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": device,
		"stats": stats,
	})
}

func (h *DeviceHandler) GetAvailableDevicesAPI(c *gin.Context) {
	devices, err := h.deviceRepo.GetAvailableDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// GetAvailableDevicesForJobAPI returns devices available for a specific job's date range
func (h *DeviceHandler) GetAvailableDevicesForJobAPI(c *gin.Context) {
	jobIDStr := c.Param("jobId")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get job details to extract dates
	// We need access to job repository for this - let me create a simple query
	var job models.Job
	// This is a bit hacky, but we need the job dates. In a better design, 
	// this would be passed as query parameters or we'd inject job repository
	db := h.deviceRepo.GetDB() // We need to add this method to device repo
	err = db.First(&job, uint(jobID)).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	devices, err := h.deviceRepo.GetAvailableDevicesForJob(uint(jobID), job.StartDate, job.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// GetDeviceTreeWithAvailability returns the device tree structure with availability checking
func (h *DeviceHandler) GetDeviceTreeWithAvailability(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	jobID := c.Query("job_id") // Optional - exclude this job from availability check
	
	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}
	
	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
		return
	}
	
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
		return
	}
	
	// Get tree data with availability information
	treeData, err := h.buildTreeDataWithAvailability(start, end, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"treeData": treeData})
}

// Hierarchical tree data structures
type TreeCategory struct {
	ID            uint              `json:"id"`
	Name          string            `json:"name"`
	DeviceCount   int               `json:"device_count"`
	DirectDevices []TreeDevice      `json:"direct_devices"`    // Devices directly in category
	Subcategories []TreeSubcategory `json:"subcategories"`
}

type TreeSubcategory struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	DeviceCount       int                      `json:"device_count"`
	DirectDevices     []TreeDevice             `json:"direct_devices"`    // Devices directly in subcategory
	Subbiercategories []TreeSubbiercategory    `json:"subbiercategories"`
}

type TreeSubbiercategory struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	DeviceCount int          `json:"device_count"`
	Devices     []TreeDevice `json:"devices"`
}

type TreeDevice struct {
	DeviceID     string `json:"device_id"`
	ProductName  string `json:"product_name"`
	SerialNumber string `json:"serial_number"`
	Status       string `json:"status"`
	Available    bool   `json:"available,omitempty"`    // Only included in availability checks
	ConflictJob  string `json:"conflict_job,omitempty"` // Job ID that conflicts
}

// buildTreeData creates a hierarchical tree structure with categories, subcategories, subbiercategories, and devices
// OPTIMIZED VERSION - Single query approach with caching to eliminate N+1 problem
func (h *DeviceHandler) buildTreeData() ([]TreeCategory, error) {
	// Check cache first
	treeCache.mutex.RLock()
	if time.Since(treeCache.timestamp) < 2*time.Minute && len(treeCache.data) > 0 {
		defer treeCache.mutex.RUnlock()
		return treeCache.data, nil
	}
	treeCache.mutex.RUnlock()
	
	
	// Get all data in ONE optimized query with preloading
	treeCategories, err := h.buildOptimizedTreeData()
	if err != nil {
		return nil, fmt.Errorf("failed to build optimized tree: %v", err)
	}
	
	// Update cache
	treeCache.mutex.Lock()
	treeCache.data = treeCategories
	treeCache.timestamp = time.Now()
	treeCache.mutex.Unlock()
	
	
	return treeCategories, nil
}

// buildTreeDataWithAvailability creates tree structure with device availability for date range
func (h *DeviceHandler) buildTreeDataWithAvailability(startDate, endDate time.Time, excludeJobID string) ([]TreeCategory, error) {
	// Get conflicting jobs for the date range first (more efficient)
	var conflictingJobs []struct {
		JobID    string `json:"job_id" gorm:"column:jobID"`
		DeviceID string `json:"device_id" gorm:"column:deviceID"`
	}
	
	query := h.deviceRepo.GetDB().
		Table("jobdevices jd").
		Select("j.jobID, jd.deviceID").
		Joins("JOIN jobs j ON jd.jobID = j.jobID").
		Where("NOT (COALESCE(j.endDate, j.startDate) < ? OR j.startDate > ?)", startDate, endDate)
	
	// Exclude current job if provided
	if excludeJobID != "" {
		query = query.Where("j.jobID != ?", excludeJobID)
	}
	
	err := query.Scan(&conflictingJobs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to check device availability: %v", err)
	}
	
	
	
	
	// Create a map for quick conflict lookup
	conflicts := make(map[string]string) // deviceID -> jobID
	for _, conflict := range conflictingJobs {
		conflicts[conflict.DeviceID] = conflict.JobID
	}
	
	// Now get tree data (after we have conflicts for better performance)
	treeCategories, err := h.buildOptimizedTreeData()
	if err != nil {
		return nil, err
	}
	
	// Update availability information in tree
	h.updateTreeAvailability(treeCategories, conflicts)
	
	return treeCategories, nil
}

// updateTreeAvailability recursively updates availability info in tree structure
func (h *DeviceHandler) updateTreeAvailability(categories []TreeCategory, conflicts map[string]string) {
	totalDevices := 0
	unavailableDevices := 0
	
	for i := range categories {
		// Update direct devices in category
		for j := range categories[i].DirectDevices {
			device := &categories[i].DirectDevices[j]
			totalDevices++
			if conflictJob, hasConflict := conflicts[device.DeviceID]; hasConflict {
				device.Available = false
				device.ConflictJob = conflictJob
				unavailableDevices++
			} else {
				device.Available = true
			}
		}
		
		// Update subcategories
		for k := range categories[i].Subcategories {
			subcategory := &categories[i].Subcategories[k]
			
			// Update direct devices in subcategory
			for j := range subcategory.DirectDevices {
				device := &subcategory.DirectDevices[j]
				if conflictJob, hasConflict := conflicts[device.DeviceID]; hasConflict {
					device.Available = false
					device.ConflictJob = conflictJob
				} else {
					device.Available = true
				}
			}
			
			// Update subbiercategories
			for l := range subcategory.Subbiercategories {
				subbiercategory := &subcategory.Subbiercategories[l]
				
				// Update devices in subbiercategory
				for j := range subbiercategory.Devices {
					device := &subbiercategory.Devices[j]
					totalDevices++
					if conflictJob, hasConflict := conflicts[device.DeviceID]; hasConflict {
						device.Available = false
						device.ConflictJob = conflictJob
						unavailableDevices++
					} else {
						device.Available = true
					}
				}
			}
		}
	}
	
}

// buildOptimizedTreeData performs a single query to get all data and builds the tree structure
func (h *DeviceHandler) buildOptimizedTreeData() ([]TreeCategory, error) {
	// Single query to get all devices with their complete hierarchy
	var devices []models.Device
	
	
	err := h.productRepo.GetDB().Model(&models.Device{}).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Joins("LEFT JOIN products ON products.productID = devices.productID").
		Joins("LEFT JOIN categories ON categories.categoryID = products.categoryID").
		Joins("LEFT JOIN subcategories ON subcategories.subcategoryID = products.subcategoryID").
		Joins("LEFT JOIN subbiercategories ON subbiercategories.subbiercategoryID = products.subbiercategoryID").
		Order("categories.name ASC, subcategories.name ASC, subbiercategories.name ASC, devices.serialnumber ASC").
		Find(&devices).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices with hierarchy: %v", err)
	}
	
	
	if len(devices) == 0 {
		return []TreeCategory{}, nil
	}
	
	// Build the tree structure from the single result set
	return h.buildTreeFromDevices(devices)
}

// buildTreeFromDevices constructs the hierarchical tree from a flat list of devices
// COMPLETELY REWRITTEN with proper nested structure building
func (h *DeviceHandler) buildTreeFromDevices(devices []models.Device) ([]TreeCategory, error) {
	
	// Group devices by their hierarchy path
	categoryGroups := make(map[uint]map[string]map[string][]models.Device)
	
	for _, device := range devices {
		if device.Product == nil || device.Product.Category == nil {
			continue
		}
		
		categoryID := device.Product.Category.CategoryID
		
		// Initialize category group if needed
		if categoryGroups[categoryID] == nil {
			categoryGroups[categoryID] = make(map[string]map[string][]models.Device)
		}
		
		var subcategoryID string = "DIRECT" // For devices directly in category
		var subbiercategoryID string = "DIRECT" // For devices directly in subcategory
		
		if device.Product.Subcategory != nil {
			subcategoryID = device.Product.Subcategory.SubcategoryID
			
			if device.Product.Subbiercategory != nil {
				subbiercategoryID = device.Product.Subbiercategory.SubbiercategoryID
			}
		}
		
		// Initialize subcategory group if needed
		if categoryGroups[categoryID][subcategoryID] == nil {
			categoryGroups[categoryID][subcategoryID] = make(map[string][]models.Device)
		}
		
		// Add device to appropriate subbiercategory
		categoryGroups[categoryID][subcategoryID][subbiercategoryID] = append(
			categoryGroups[categoryID][subcategoryID][subbiercategoryID], device)
	}
	
	// Build the tree structure
	var treeCategories []TreeCategory
	
	for categoryID, subcategoryGroups := range categoryGroups {
		// Find the category info from first device
		var categoryName string
		for _, subGroup := range subcategoryGroups {
			for _, deviceList := range subGroup {
				if len(deviceList) > 0 && deviceList[0].Product != nil && deviceList[0].Product.Category != nil {
					categoryName = deviceList[0].Product.Category.Name
					break
				}
			}
			if categoryName != "" {
				break
			}
		}
		
		treeCategory := TreeCategory{
			ID:            categoryID,
			Name:          categoryName,
			DeviceCount:   0,
			DirectDevices: []TreeDevice{},
			Subcategories: []TreeSubcategory{},
		}
		
		
		for subcategoryID, subbiercategoryGroups := range subcategoryGroups {
			if subcategoryID == "DIRECT" {
				// Devices directly in category (no subcategory)
				if deviceList, exists := subbiercategoryGroups["DIRECT"]; exists {
					for _, device := range deviceList {
						treeCategory.DirectDevices = append(treeCategory.DirectDevices, h.convertToTreeDevice(device))
						treeCategory.DeviceCount++
					}
				}
			} else {
				// Build subcategory
				var subcategoryName string
				var totalDevicesInSubcategory int
				
				// Find subcategory name from first device
				for _, deviceList := range subbiercategoryGroups {
					if len(deviceList) > 0 && deviceList[0].Product != nil && deviceList[0].Product.Subcategory != nil {
						subcategoryName = deviceList[0].Product.Subcategory.Name
						break
					}
				}
				
				treeSubcategory := TreeSubcategory{
					ID:                subcategoryID,
					Name:              subcategoryName,
					DeviceCount:       0,
					DirectDevices:     []TreeDevice{},
					Subbiercategories: []TreeSubbiercategory{},
				}
				
				
				for subbiercategoryID, deviceList := range subbiercategoryGroups {
					if subbiercategoryID == "DIRECT" {
						// Devices directly in subcategory (no subbiercategory)
						for _, device := range deviceList {
							treeSubcategory.DirectDevices = append(treeSubcategory.DirectDevices, h.convertToTreeDevice(device))
							treeSubcategory.DeviceCount++
							totalDevicesInSubcategory++
						}
					} else {
						// Build subbiercategory
						var subbiercategoryName string
						if len(deviceList) > 0 && deviceList[0].Product != nil && deviceList[0].Product.Subbiercategory != nil {
							subbiercategoryName = deviceList[0].Product.Subbiercategory.Name
						}
						
						var treeDevices []TreeDevice
						for _, device := range deviceList {
							treeDevices = append(treeDevices, h.convertToTreeDevice(device))
						}
						
						treeSubbiercategory := TreeSubbiercategory{
							ID:          subbiercategoryID,
							Name:        subbiercategoryName,
							DeviceCount: len(treeDevices),
							Devices:     treeDevices,
						}
						
						treeSubcategory.Subbiercategories = append(treeSubcategory.Subbiercategories, treeSubbiercategory)
						treeSubcategory.DeviceCount += len(treeDevices)
						totalDevicesInSubcategory += len(treeDevices)
						
					}
				}
				
				// Sort subbiercategories by name
				sort.Slice(treeSubcategory.Subbiercategories, func(i, j int) bool {
					return treeSubcategory.Subbiercategories[i].Name < treeSubcategory.Subbiercategories[j].Name
				})
				
				treeCategory.Subcategories = append(treeCategory.Subcategories, treeSubcategory)
				treeCategory.DeviceCount += totalDevicesInSubcategory
			}
		}
		
		// Sort subcategories by name
		sort.Slice(treeCategory.Subcategories, func(i, j int) bool {
			return treeCategory.Subcategories[i].Name < treeCategory.Subcategories[j].Name
		})
		
		treeCategories = append(treeCategories, treeCategory)
	}
	
	// Sort categories by name
	sort.Slice(treeCategories, func(i, j int) bool {
		return treeCategories[i].Name < treeCategories[j].Name
	})
	
	
	
	return treeCategories, nil
}

// Helper function to convert Device to TreeDevice
func (h *DeviceHandler) convertToTreeDevice(device models.Device) TreeDevice {
	serialNum := ""
	if device.SerialNumber != nil {
		serialNum = *device.SerialNumber
	}
	
	productName := "Unknown Product"
	if device.Product != nil && device.Product.Name != "" {
		productName = device.Product.Name
	}
	
	return TreeDevice{
		DeviceID:     device.DeviceID,
		ProductName:  productName,
		SerialNumber: serialNum,
		Status:       device.Status,
	}
}

// Helper function to get devices directly in category (without subcategory)
func (h *DeviceHandler) getDirectCategoryDevices(categoryID uint) ([]models.DeviceWithJobInfo, error) {
	// For now, return empty slice - we'll focus on the hierarchical structure first
	// Direct category devices are rare in most setups
	return []models.DeviceWithJobInfo{}, nil
}