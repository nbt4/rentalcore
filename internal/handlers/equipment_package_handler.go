package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type EquipmentPackageHandler struct {
	packageRepo *repository.EquipmentPackageRepository
	deviceRepo  *repository.DeviceRepository
}

func NewEquipmentPackageHandler(packageRepo *repository.EquipmentPackageRepository, deviceRepo *repository.DeviceRepository) *EquipmentPackageHandler {
	return &EquipmentPackageHandler{
		packageRepo: packageRepo,
		deviceRepo:  deviceRepo,
	}
}

// Equipment Package Templates and Forms
func (h *EquipmentPackageHandler) ShowPackagesList(c *gin.Context) {
	log.Printf("🎯 EQUIPMENT PACKAGE HANDLER: ShowPackagesList called")
	// Parse filter parameters
	params := parseFilterParams(c)
	
	packages, err := h.packageRepo.List(params)
	if err != nil {
		log.Printf("Error fetching equipment packages: %v", err)
		c.HTML(http.StatusInternalServerError, "error_page.html", gin.H{
			"error": "Failed to load equipment packages",
		})
		return
	}

	// Calculate total values and device counts for display
	for i := range packages {
		log.Printf("🎯 BEFORE ENRICH: Package %d ('%s') has %d PackageDevices", 
			packages[i].PackageID, packages[i].Name, len(packages[i].PackageDevices))
		h.enrichPackageData(&packages[i])
		log.Printf("🎯 AFTER ENRICH: Package %d ('%s') has %d PackageDevices and DeviceCount=%d", 
			packages[i].PackageID, packages[i].Name, len(packages[i].PackageDevices), packages[i].DeviceCount)
	}

	// Get total count for pagination
	totalCount, _ := h.packageRepo.GetTotalCount(params)

	// Get popular packages for dashboard
	popularPackages, _ := h.packageRepo.GetPopularPackages(5)

	// Debug template data before rendering
	log.Printf("🎯 TEMPLATE DEBUG: Rendering with %d packages", len(packages))
	for i, pkg := range packages {
		log.Printf("🎯 TEMPLATE DEBUG: Package %d: ID=%d, Name='%s', PackageDevices=%d, DeviceCount=%d", 
			i, pkg.PackageID, pkg.Name, len(pkg.PackageDevices), pkg.DeviceCount)
	}
	
	user, _ := GetCurrentUser(c)
	
	c.HTML(http.StatusOK, "equipment_packages_standalone.html", gin.H{
		"packages":        packages,
		"popularPackages": popularPackages,
		"totalCount":      totalCount,
		"filters":         params,
		"user":            user,
		"currentPage":     "packages",
	})
}

func (h *EquipmentPackageHandler) ShowPackageForm(c *gin.Context) {
	packageID := c.Param("id")
	
	// Get available devices
	availableDevices, err := h.packageRepo.GetAvailableDevices()
	if err != nil {
		log.Printf("Error fetching available devices: %v", err)
		c.HTML(http.StatusInternalServerError, "error_page.html", gin.H{
			"error": "Failed to load available devices",
		})
		return
	}

	// Get all categories for categorization
	categories := h.getPackageCategories()

	var pkg *models.EquipmentPackage
	if packageID != "" && packageID != "new" {
		id, err := strconv.ParseUint(packageID, 10, 32)
		if err == nil {
			pkg, _ = h.packageRepo.GetByID(uint(id))
		}
	}

	c.HTML(http.StatusOK, "equipment_package_form.html", gin.H{
		"package":          pkg,
		"availableDevices": availableDevices,
		"categories":       categories,
	})
}

func (h *EquipmentPackageHandler) ShowPackageDetail(c *gin.Context) {
	packageID := c.Param("id")
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error_page.html", gin.H{
			"error": "Invalid package ID",
		})
		return
	}

	pkg, err := h.packageRepo.GetByIDWithDeviceDetails(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error_page.html", gin.H{
			"error": "Package not found",
		})
		return
	}

	// Enrich package with calculated data
	h.enrichPackageData(pkg)

	// Get package statistics
	stats, _ := h.packageRepo.GetPackageStats(uint(id))

	// Validate package devices
	isValid, invalidDevices, _ := h.packageRepo.ValidatePackageDevices(uint(id))

	c.HTML(http.StatusOK, "equipment_package_detail.html", gin.H{
		"package":        pkg,
		"stats":          stats,
		"isValid":        isValid,
		"invalidDevices": invalidDevices,
	})
}

// API Endpoints
func (h *EquipmentPackageHandler) GetPackages(c *gin.Context) {
	log.Printf("GetPackages called")
	params := parseFilterParams(c)
	
	packages, err := h.packageRepo.List(params)
	if err != nil {
		log.Printf("Error fetching packages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Found %d packages", len(packages))
	// Enrich packages with calculated data
	for i := range packages {
		h.enrichPackageData(&packages[i])
	}

	totalCount, _ := h.packageRepo.GetTotalCount(params)

	c.JSON(http.StatusOK, gin.H{
		"packages":   packages,
		"totalCount": totalCount,
		"page":       params.Page,
		"pageSize":   params.Limit,
	})
}

func (h *EquipmentPackageHandler) GetPackage(c *gin.Context) {
	packageID := c.Param("id")
	log.Printf("GetPackage called with packageID: %s", packageID)
	
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		log.Printf("Invalid package ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	pkg, err := h.packageRepo.GetByIDWithDeviceDetails(uint(id))
	if err != nil {
		log.Printf("Package not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	// Enrich package with calculated data
	h.enrichPackageData(pkg)

	// Get package statistics
	stats, _ := h.packageRepo.GetPackageStats(uint(id))

	// Validate package devices
	isValid, invalidDevices, _ := h.packageRepo.ValidatePackageDevices(uint(id))

	log.Printf("Successfully returning package data for ID: %d", id)
	c.JSON(http.StatusOK, gin.H{
		"package":        pkg,
		"stats":          stats,
		"isValid":        isValid,
		"invalidDevices": invalidDevices,
	})
}

func (h *EquipmentPackageHandler) CreatePackage(c *gin.Context) {
	var req models.CreateEquipmentPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate devices exist and are available
	if err := h.validatePackageDevices(req.Devices); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create package
	pkg := &models.EquipmentPackage{
		Name:            req.Name,
		Description:     req.Description,
		PackagePrice:    req.PackagePrice,
		DiscountPercent: req.DiscountPercent,
		MinRentalDays:   req.MinRentalDays,
		MaxRentalDays:   req.MaxRentalDays,
		IsActive:        req.IsActive,
		Category:        req.Category,
		Tags:            req.Tags,
		PackageItems:    json.RawMessage("[]"),
	}

	// Convert device requests to package devices
	var deviceMappings []models.PackageDevice
	for _, deviceReq := range req.Devices {
		deviceMappings = append(deviceMappings, models.PackageDevice{
			DeviceID:    deviceReq.DeviceID,
			Quantity:    deviceReq.Quantity,
			CustomPrice: deviceReq.CustomPrice,
			IsRequired:  deviceReq.IsRequired,
			Notes:       deviceReq.Notes,
			SortOrder:   deviceReq.SortOrder,
		})
	}

	if err := h.packageRepo.CreateWithDevices(pkg, deviceMappings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich the created package
	h.enrichPackageData(pkg)

	c.JSON(http.StatusCreated, gin.H{"package": pkg})
}

func (h *EquipmentPackageHandler) UpdatePackage(c *gin.Context) {
	packageID := c.Param("id")
	log.Printf("🔄 UpdatePackage called with packageID: %s", packageID)
	
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		log.Printf("Invalid package ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	// Log the raw request body
	bodyBytes, _ := c.GetRawData()
	log.Printf("Raw request body: %s", string(bodyBytes))
	
	// Reset the request body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	
	var req models.UpdateEquipmentPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		log.Printf("Raw JSON was: %s", string(bodyBytes))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	log.Printf("Update request data: %+v", req)

	// Get existing package
	pkg, err := h.packageRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	// Skip validation for updates - devices are being managed through associations
	// Validation is only needed for new packages, not for updates
	// if err := h.validatePackageDevices(convertUpdateToCreateDevices(req.Devices)); err != nil {
	// 	log.Printf("Device validation failed: %v", err)
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	// Update package fields
	pkg.Name = req.Name
	pkg.Description = req.Description
	pkg.PackagePrice = req.PackagePrice
	pkg.DiscountPercent = req.DiscountPercent
	pkg.MinRentalDays = req.MinRentalDays
	pkg.MaxRentalDays = req.MaxRentalDays
	pkg.IsActive = req.IsActive
	pkg.Category = req.Category
	pkg.Tags = req.Tags

	// Update package
	if err := h.packageRepo.Update(pkg); err != nil {
		log.Printf("Package update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update device associations
	var deviceMappings []models.PackageDevice
	log.Printf("🔄 Building device mappings for %d devices", len(req.Devices))
	for _, deviceReq := range req.Devices {
		log.Printf("🔄 Adding device mapping: %s (quantity: %d)", deviceReq.DeviceID, deviceReq.Quantity)
		
		// Validate device exists before adding to mappings
		_, err := h.deviceRepo.GetByID(deviceReq.DeviceID)
		if err != nil {
			log.Printf("❌ Device %s does not exist or is not accessible - skipping", deviceReq.DeviceID)
			continue
		}
		
		deviceMappings = append(deviceMappings, models.PackageDevice{
			DeviceID:    deviceReq.DeviceID,
			Quantity:    deviceReq.Quantity,
			CustomPrice: deviceReq.CustomPrice,
			IsRequired:  deviceReq.IsRequired,
			Notes:       deviceReq.Notes,
			SortOrder:   deviceReq.SortOrder,
		})
	}

	if err := h.packageRepo.UpdateDeviceAssociations(uint(id), deviceMappings); err != nil {
		log.Printf("Device association update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload package with updated data (without device preload to prevent auto-creation)
	pkg, _ = h.packageRepo.GetByIDWithoutDevicePreload(uint(id))
	h.enrichPackageData(pkg)

	c.JSON(http.StatusOK, gin.H{"package": pkg})
}

func (h *EquipmentPackageHandler) DeletePackage(c *gin.Context) {
	packageID := c.Param("id")
	log.Printf("DeletePackage called with packageID: %s", packageID)
	
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		log.Printf("Invalid package ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	if err := h.packageRepo.Delete(uint(id)); err != nil {
		log.Printf("Failed to delete package: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Package deleted successfully: %d", id)
	c.JSON(http.StatusOK, gin.H{"message": "Package deleted successfully"})
}

// Advanced Features
func (h *EquipmentPackageHandler) ClonePackage(c *gin.Context) {
	packageID := c.Param("id")
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	originalPkg, err := h.packageRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	// Create cloned package
	clonedPkg := &models.EquipmentPackage{
		Name:            originalPkg.Name + " (Copy)",
		Description:     originalPkg.Description,
		PackagePrice:    originalPkg.PackagePrice,
		DiscountPercent: originalPkg.DiscountPercent,
		MinRentalDays:   originalPkg.MinRentalDays,
		MaxRentalDays:   originalPkg.MaxRentalDays,
		IsActive:        false, // Start as inactive
		Category:        originalPkg.Category,
		Tags:            originalPkg.Tags,
		PackageItems:    originalPkg.PackageItems,
	}

	// Clone device associations
	var deviceMappings []models.PackageDevice
	for _, device := range originalPkg.PackageDevices {
		deviceMappings = append(deviceMappings, models.PackageDevice{
			DeviceID:    device.DeviceID,
			Quantity:    device.Quantity,
			CustomPrice: device.CustomPrice,
			IsRequired:  device.IsRequired,
			Notes:       device.Notes,
			SortOrder:   device.SortOrder,
		})
	}

	if err := h.packageRepo.CreateWithDevices(clonedPkg, deviceMappings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.enrichPackageData(clonedPkg)

	c.JSON(http.StatusCreated, gin.H{"package": clonedPkg})
}

func (h *EquipmentPackageHandler) SearchPackages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	params := parseFilterParams(c)
	packages, err := h.packageRepo.Search(query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich packages
	for i := range packages {
		h.enrichPackageData(&packages[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"packages": packages,
		"query":    query,
		"count":    len(packages),
	})
}

func (h *EquipmentPackageHandler) GetPackageCategories(c *gin.Context) {
	categories := h.getPackageCategories()
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *EquipmentPackageHandler) GetPopularPackages(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	packages, err := h.packageRepo.GetPopularPackages(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range packages {
		h.enrichPackageData(&packages[i])
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}

func (h *EquipmentPackageHandler) ValidatePackage(c *gin.Context) {
	packageID := c.Param("id")
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	isValid, invalidDevices, err := h.packageRepo.ValidatePackageDevices(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"isValid":        isValid,
		"invalidDevices": invalidDevices,
	})
}

func (h *EquipmentPackageHandler) GetPackageStats(c *gin.Context) {
	packageID := c.Param("id")
	id, err := strconv.ParseUint(packageID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	stats, err := h.packageRepo.GetPackageStats(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (h *EquipmentPackageHandler) GetAvailableDevices(c *gin.Context) {
	devices, err := h.packageRepo.GetAvailableDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// Bulk Operations
func (h *EquipmentPackageHandler) BulkUpdatePackages(c *gin.Context) {
	var req struct {
		PackageIDs []uint `json:"packageIds" binding:"required"`
		Updates    struct {
			IsActive        *bool    `json:"isActive"`
			Category        *string  `json:"category"`
			DiscountPercent *float64 `json:"discountPercent"`
		} `json:"updates" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedCount := 0
	errors := []string{}

	for _, packageID := range req.PackageIDs {
		pkg, err := h.packageRepo.GetByID(packageID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Package %d not found", packageID))
			continue
		}

		// Apply updates
		if req.Updates.IsActive != nil {
			pkg.IsActive = *req.Updates.IsActive
		}
		if req.Updates.Category != nil {
			pkg.Category = *req.Updates.Category
		}
		if req.Updates.DiscountPercent != nil {
			pkg.DiscountPercent = *req.Updates.DiscountPercent
		}

		if err := h.packageRepo.Update(pkg); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to update package %d: %v", packageID, err))
			continue
		}

		updatedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"updatedCount": updatedCount,
		"errors":       errors,
	})
}

// Helper Functions
func (h *EquipmentPackageHandler) enrichPackageData(pkg *models.EquipmentPackage) {
	// Calculate total value and price
	totalValue := 0.0
	calculatedPrice := 0.0

	for _, device := range pkg.PackageDevices {
		if device.Device != nil && device.Device.Product != nil {
			devicePrice := 0.0
			if device.CustomPrice != nil {
				devicePrice = *device.CustomPrice
			} else if device.Device.Product.ItemCostPerDay != nil {
				devicePrice = *device.Device.Product.ItemCostPerDay
			}
			
			totalValue += devicePrice * float64(device.Quantity)
			calculatedPrice += devicePrice * float64(device.Quantity)
		}
	}

	// Apply package discount
	if pkg.DiscountPercent > 0 {
		calculatedPrice = calculatedPrice * (1 - pkg.DiscountPercent/100)
	}

	// Use package price if set, otherwise use calculated price
	if pkg.PackagePrice != nil {
		calculatedPrice = *pkg.PackagePrice
	}

	pkg.TotalValue = totalValue
	pkg.CalculatedPrice = calculatedPrice
	
	// Only update DeviceCount if PackageDevices is populated
	// For list views, DeviceCount is set by repository and PackageDevices is empty for performance
	if len(pkg.PackageDevices) > 0 {
		pkg.DeviceCount = len(pkg.PackageDevices)
	}
}

func (h *EquipmentPackageHandler) validatePackageDevices(devices []models.CreatePackageDeviceRequest) error {
	log.Printf("🔍 VALIDATION: Starting validation for %d devices", len(devices))
	for _, device := range devices {
		log.Printf("🔍 VALIDATION: Validating device %s", device.DeviceID)
		// Check if device exists and is available
		existingDevice, err := h.deviceRepo.GetByID(device.DeviceID)
		if err != nil {
			log.Printf("❌ VALIDATION: Device %s not found: %v", device.DeviceID, err)
			return fmt.Errorf("device %s not found", device.DeviceID)
		}

		log.Printf("✅ VALIDATION: Device %s exists with status: %s", device.DeviceID, existingDevice.Status)
		if existingDevice.Status != "free" && existingDevice.Status != "available" && existingDevice.Status != "ready" {
			log.Printf("❌ VALIDATION: Device %s is not available (status: %s)", device.DeviceID, existingDevice.Status)
			return fmt.Errorf("device %s is not available (status: %s)", device.DeviceID, existingDevice.Status)
		}
	}
	log.Printf("✅ VALIDATION: All devices validated successfully")
	return nil
}

func (h *EquipmentPackageHandler) getPackageCategories() []string {
	return []string{
		"Audio/Video Equipment",
		"Lighting Equipment", 
		"Sound Systems",
		"Stage Equipment",
		"DJ Equipment",
		"Photo/Video Production",
		"Event Furniture",
		"Decoration",
		"Security Equipment",
		"Power & Electrical",
		"Transport & Logistics",
		"Catering Equipment",
		"Custom Packages",
	}
}

func parseFilterParams(c *gin.Context) *models.FilterParams {
	params := &models.FilterParams{
		SearchTerm: c.Query("search"),
		Category:   c.Query("category"),
		SortBy:     c.DefaultQuery("sort_by", "created_at"),
		SortOrder:  c.DefaultQuery("sort_order", "desc"),
		Limit:      25,
		Offset:     0,
		Page:       1,
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
			params.Offset = (p - 1) * params.Limit
		}
	}

	return params
}

func convertUpdateToCreateDevices(updateDevices []models.UpdatePackageDeviceRequest) []models.CreatePackageDeviceRequest {
	var createDevices []models.CreatePackageDeviceRequest
	for _, device := range updateDevices {
		createDevices = append(createDevices, models.CreatePackageDeviceRequest{
			DeviceID:    device.DeviceID,
			Quantity:    device.Quantity,
			CustomPrice: device.CustomPrice,
			IsRequired:  device.IsRequired,
			Notes:       device.Notes,
			SortOrder:   device.SortOrder,
		})
	}
	return createDevices
}