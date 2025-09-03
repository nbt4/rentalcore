package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"gorm.io/gorm"
)

type WorkflowHandler struct {
	jobRepo        *repository.JobRepository
	customerRepo   *repository.CustomerRepository
	packageRepo    *repository.EquipmentPackageRepository
	deviceRepo     *repository.DeviceRepository
	db             *gorm.DB
	barcodeService *services.BarcodeService
}

func NewWorkflowHandler(jobRepo *repository.JobRepository, customerRepo *repository.CustomerRepository, packageRepo *repository.EquipmentPackageRepository, deviceRepo *repository.DeviceRepository, db *gorm.DB, barcodeService *services.BarcodeService) *WorkflowHandler {
	return &WorkflowHandler{
		jobRepo:        jobRepo,
		customerRepo:   customerRepo,
		packageRepo:    packageRepo,
		deviceRepo:     deviceRepo,
		db:             db,
		barcodeService: barcodeService,
	}
}

// ================================================================
// HELPER FUNCTIONS
// ================================================================

// getDefaultStatusID returns the default status ID for new jobs
// TODO: This should be configurable or fetched from database
func getDefaultStatusID() *uint {
	// For now, return nil to let the job creation handle the default
	// This prevents the hardcoded status ID issue
	return nil
}

// ================================================================
// EQUIPMENT PACKAGES - PLACEHOLDER METHODS
// ================================================================

// ListEquipmentPackages displays all equipment packages
func (h *WorkflowHandler) ListEquipmentPackages(c *gin.Context) {
	log.Printf("🎯 WORKFLOW HANDLER: ListEquipmentPackages called")
	user, _ := GetCurrentUser(c)
	
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	packages, err := h.packageRepo.List(params)
	if err != nil {
		log.Printf("ListEquipmentPackages: Error fetching packages: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to load equipment packages", "user": user})
		return
	}

	log.Printf("🎯 WORKFLOW HANDLER: Got %d packages from repository", len(packages))

	// Use the same enrichment logic as equipment package handler
	for i := range packages {
		log.Printf("🎯 WORKFLOW: Package %d ('%s') has %d PackageDevices BEFORE enrichment", 
			packages[i].PackageID, packages[i].Name, len(packages[i].PackageDevices))
		// Calculate total value and price
		totalValue := 0.0
		calculatedPrice := 0.0
		for _, device := range packages[i].PackageDevices {
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
		if packages[i].DiscountPercent > 0 {
			calculatedPrice = calculatedPrice * (1 - packages[i].DiscountPercent/100)
		}
		// Use package price if set, otherwise use calculated price
		if packages[i].PackagePrice != nil {
			calculatedPrice = *packages[i].PackagePrice
		}
		packages[i].TotalValue = totalValue
		packages[i].CalculatedPrice = calculatedPrice
		packages[i].DeviceCount = len(packages[i].PackageDevices)
		log.Printf("🎯 WORKFLOW: Package %d ('%s') has %d PackageDevices AFTER enrichment, DeviceCount=%d", 
			packages[i].PackageID, packages[i].Name, len(packages[i].PackageDevices), packages[i].DeviceCount)
	}

	// Get additional data that the standalone template expects
	totalCount, _ := h.packageRepo.GetTotalCount(params)
	popularPackages, _ := h.packageRepo.GetPopularPackages(5)

	log.Printf("DEBUG: ListEquipmentPackages: Attempting to render equipment_packages_standalone.html")
	c.HTML(http.StatusOK, "equipment_packages_standalone.html", gin.H{
		"packages":        packages,
		"popularPackages": popularPackages,
		"totalCount":      totalCount,
		"filters":         params,
		"user":            user,
	})
}

// NewEquipmentPackageForm displays the form for creating a new equipment package
func (h *WorkflowHandler) NewEquipmentPackageForm(c *gin.Context) {
	// Get current user for base template
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		log.Printf("NewEquipmentPackageForm: User not authenticated")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get available devices for the dropdown
	availableDevices, err := h.packageRepo.GetAvailableDevices()
	if err != nil {
		log.Printf("NewEquipmentPackageForm: Error fetching available devices: %v", err)
		availableDevices = []models.Device{} // Use empty slice if error
	}
	
	log.Printf("NewEquipmentPackageForm: Found %d available devices", len(availableDevices))
	if len(availableDevices) > 0 {
		log.Printf("NewEquipmentPackageForm: Sample device: ID=%s, Product=%v", 
			availableDevices[0].DeviceID, 
			func() string { if availableDevices[0].Product != nil { return availableDevices[0].Product.Name } else { return "nil" } }())
	}

	c.HTML(http.StatusOK, "equipment_package_form.html", gin.H{
		"title":            "New Equipment Package",
		"package":          nil,
		"isEdit":           false,
		"user":             currentUser,
		"availableDevices": availableDevices,
	})
}

// CreateEquipmentPackage creates a new equipment package
func (h *WorkflowHandler) CreateEquipmentPackage(c *gin.Context) {
	// Get current user
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		availableDevices, _ := h.packageRepo.GetAvailableDevices()
		c.HTML(http.StatusUnauthorized, "equipment_package_form.html", gin.H{
			"title":            "New Equipment Package",
			"package":          &models.EquipmentPackage{},
			"isEdit":           false,
			"error":            "Authentication required. Please log in to continue.",
			"user":             currentUser,
			"availableDevices": availableDevices,
		})
		return
	}

	// Parse form data
	var pkg models.EquipmentPackage
	pkg.Name = c.PostForm("name")
	pkg.Description = c.PostForm("description")
	
	// Parse optional fields
	if packagePrice := c.PostForm("packagePrice"); packagePrice != "" {
		if price, err := strconv.ParseFloat(packagePrice, 64); err == nil {
			pkg.PackagePrice = &price
		}
	}
	
	if discountPercent := c.PostForm("discountPercent"); discountPercent != "" {
		if discount, err := strconv.ParseFloat(discountPercent, 64); err == nil {
			pkg.DiscountPercent = discount
		}
	}
	
	if minRentalDays := c.PostForm("minRentalDays"); minRentalDays != "" {
		if days, err := strconv.Atoi(minRentalDays); err == nil {
			pkg.MinRentalDays = days
		}
	}
	
	pkg.IsActive = c.PostForm("isActive") == "on"
	
	// Handle package items
	packageItemsStr := c.PostForm("packageItems")
	if packageItemsStr == "" {
		packageItemsStr = "[]"
	}
	pkg.PackageItems = json.RawMessage(packageItemsStr)

	// Parse device selections
	var deviceMappings []models.PackageDevice
	
	// Parse devices array from form
	i := 0
	for {
		deviceIDKey := "devices[" + strconv.Itoa(i) + "][deviceID]"
		deviceID := c.PostForm(deviceIDKey)
		
		if deviceID == "" {
			break // No more devices
		}
		
		// Parse device data
		quantityKey := "devices[" + strconv.Itoa(i) + "][quantity]"
		customPriceKey := "devices[" + strconv.Itoa(i) + "][customPrice]"
		isRequiredKey := "devices[" + strconv.Itoa(i) + "][isRequired]"
		notesKey := "devices[" + strconv.Itoa(i) + "][notes]"
		
		quantity, _ := strconv.ParseUint(c.PostForm(quantityKey), 10, 32)
		if quantity == 0 {
			quantity = 1
		}
		
		var customPrice *float64
		if customPriceStr := c.PostForm(customPriceKey); customPriceStr != "" {
			if price, err := strconv.ParseFloat(customPriceStr, 64); err == nil {
				customPrice = &price
			}
		}
		
		isRequired := c.PostForm(isRequiredKey) == "true"
		notes := c.PostForm(notesKey)
		
		deviceMapping := models.PackageDevice{
			DeviceID:    deviceID,
			Quantity:    uint(quantity),
			CustomPrice: customPrice,
			IsRequired:  isRequired,
			Notes:       notes,
		}
		
		deviceMappings = append(deviceMappings, deviceMapping)
		i++
	}

	// Validate required fields
	if pkg.Name == "" {
		availableDevices, _ := h.packageRepo.GetAvailableDevices()
		c.HTML(http.StatusBadRequest, "equipment_package_form.html", gin.H{
			"title":            "New Equipment Package",
			"package":          &pkg,
			"isEdit":           false,
			"error":            "Package name is required",
			"user":             currentUser,
			"availableDevices": availableDevices,
		})
		return
	}

	// Set creator
	pkg.CreatedBy = &currentUser.UserID
	
	// Save to database with device associations
	if err := h.packageRepo.CreateWithDevices(&pkg, deviceMappings); err != nil {
		log.Printf("CreateEquipmentPackage: Database error: %v", err)
		availableDevices, _ := h.packageRepo.GetAvailableDevices()
		c.HTML(http.StatusInternalServerError, "equipment_package_form.html", gin.H{
			"title":            "New Equipment Package",
			"package":          &pkg,
			"isEdit":           false,
			"error":            "Failed to create equipment package: " + err.Error(),
			"user":             currentUser,
			"availableDevices": availableDevices,
		})
		return
	}
	
	log.Printf("CreateEquipmentPackage: Successfully created package '%s' (ID: %d) with %d devices by user %s", 
		pkg.Name, pkg.PackageID, len(deviceMappings), currentUser.Username)
	
	// Redirect to packages list on success
	c.Redirect(http.StatusSeeOther, "/workflow/packages")
}

// GetEquipmentPackage returns a specific equipment package
func (h *WorkflowHandler) GetEquipmentPackage(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	packageIDStr := c.Param("id")
	packageID, err := strconv.ParseUint(packageIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid package ID", "user": user})
		return
	}

	pkg, err := h.packageRepo.GetByID(uint(packageID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Equipment package not found", "user": user})
		return
	}

	var calculatedPrice float64
	for _, item := range pkg.PackageDevices {
		if item.CustomPrice != nil {
			calculatedPrice += *item.CustomPrice * float64(item.Quantity)
		} else if item.Device != nil && item.Device.Product != nil && item.Device.Product.ItemCostPerDay != nil {
			calculatedPrice += *item.Device.Product.ItemCostPerDay * float64(item.Quantity)
		}
	}
	pkg.CalculatedPrice = calculatedPrice

	if pkg.PackagePrice != nil {
		pkg.TotalValue = *pkg.PackagePrice
	} else {
		pkg.TotalValue = calculatedPrice
	}

	c.HTML(http.StatusOK, "equipment_package_detail.html", gin.H{
		"title":   "Package Details",
		"package": pkg,
		"user":    user,
	})
}

func (h *WorkflowHandler) GetEquipmentPackageForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	packageIDStr := c.Param("id")

	if packageIDStr == "new" {
		availableDevices, err := h.packageRepo.GetAvailableDevices()
		if err != nil {
			log.Printf("GetEquipmentPackageForm: Error fetching available devices: %v", err)
		}
		c.HTML(http.StatusOK, "equipment_package_form.html", gin.H{
			"title":            "New Equipment Package",
			"package":          nil,
			"isEdit":           false,
			"user":             user,
			"availableDevices": availableDevices,
		})
		return
	}

	packageID, err := strconv.ParseUint(packageIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid package ID", "user": user})
		return
	}

	pkg, err := h.packageRepo.GetByID(uint(packageID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Equipment package not found", "user": user})
		return
	}

	availableDevices, err := h.packageRepo.GetAvailableDevices()
	if err != nil {
		log.Printf("GetEquipmentPackageForm: Error fetching available devices: %v", err)
	}

	c.HTML(http.StatusOK, "equipment_package_form.html", gin.H{
		"title":            "Edit Equipment Package",
		"package":          pkg,
		"isEdit":           true,
		"user":             user,
		"availableDevices": availableDevices,
	})
}

// UpdateEquipmentPackage updates an existing equipment package via API
func (h *WorkflowHandler) UpdateEquipmentPackage(c *gin.Context) {
	packageIDStr := c.Param("id")
	packageID, err := strconv.ParseUint(packageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	// Get current user
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Find existing package
	var pkg models.EquipmentPackage
	if err := h.db.Where("packageID = ?", packageID).First(&pkg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Parse form data
	pkg.Name = c.PostForm("name")
	pkg.Description = c.PostForm("description")

	// Parse optional fields
	if packagePrice := c.PostForm("packagePrice"); packagePrice != "" {
		if price, err := strconv.ParseFloat(packagePrice, 64); err == nil {
			pkg.PackagePrice = &price
		}
	} else {
		pkg.PackagePrice = nil
	}

	if discountPercent := c.PostForm("discountPercent"); discountPercent != "" {
		if discount, err := strconv.ParseFloat(discountPercent, 64); err == nil {
			pkg.DiscountPercent = discount
		}
	}

	if minRentalDays := c.PostForm("minRentalDays"); minRentalDays != "" {
		if days, err := strconv.Atoi(minRentalDays); err == nil {
			pkg.MinRentalDays = days
		}
	}

	pkg.IsActive = c.PostForm("isActive") == "on"

	// Handle package items
	packageItemsStr := c.PostForm("packageItems")
	if packageItemsStr == "" {
		packageItemsStr = "[]"
	}
	pkg.PackageItems = json.RawMessage(packageItemsStr)

	// Parse device selections
	var deviceMappings []models.PackageDevice

	// Parse devices array from form
	i := 0
	for {
		deviceIDKey := "devices[" + strconv.Itoa(i) + "][deviceID]"
		deviceID := c.PostForm(deviceIDKey)

		if deviceID == "" {
			break // No more devices
		}

		// Parse device data
		quantityKey := "devices[" + strconv.Itoa(i) + "][quantity]"
		customPriceKey := "devices[" + strconv.Itoa(i) + "][customPrice]"
		isRequiredKey := "devices[" + strconv.Itoa(i) + "][isRequired]"
		notesKey := "devices[" + strconv.Itoa(i) + "][notes]"

		quantity, _ := strconv.ParseUint(c.PostForm(quantityKey), 10, 32)
		if quantity == 0 {
			quantity = 1
		}

		var customPrice *float64
		if customPriceStr := c.PostForm(customPriceKey); customPriceStr != "" {
			if price, err := strconv.ParseFloat(customPriceStr, 64); err == nil {
				customPrice = &price
			}
		}

		isRequired := c.PostForm(isRequiredKey) == "true"
		notes := c.PostForm(notesKey)

		deviceMapping := models.PackageDevice{
			DeviceID:    deviceID,
			Quantity:    uint(quantity),
			CustomPrice: customPrice,
			IsRequired:  isRequired,
			Notes:       notes,
		}

		deviceMappings = append(deviceMappings, deviceMapping)
		i++
	}

	// Update device associations
	if err := h.packageRepo.UpdateDeviceAssociations(uint(packageID), deviceMappings); err != nil {
		log.Printf("UpdateEquipmentPackage: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device associations: " + err.Error()})
		return
	}

	// Save changes to the package
	if err := h.packageRepo.Update(&pkg); err != nil {
		log.Printf("UpdateEquipmentPackage: Error updating package %d: %v", packageID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update package"})
		return
	}

	log.Printf("UpdateEquipmentPackage: Package %d updated successfully by user %s", packageID, currentUser.Username)
	c.Redirect(http.StatusSeeOther, "/workflow/packages")
}

// DeleteEquipmentPackage deletes an equipment package
func (h *WorkflowHandler) DeleteEquipmentPackage(c *gin.Context) {
	packageIDStr := c.Param("id")
	packageID, err := strconv.ParseUint(packageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	// Get current user
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Find existing package
	var pkg models.EquipmentPackage
	if err := h.db.Where("packageID = ?", packageID).First(&pkg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Delete associated package devices first
	if err := h.db.Where("packageID = ?", packageID).Delete(&models.PackageDevice{}).Error; err != nil {
		log.Printf("DeleteEquipmentPackage: Error deleting package devices for package %d: %v", packageID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete package devices"})
		return
	}

	// Delete the package
	if err := h.db.Delete(&pkg).Error; err != nil {
		log.Printf("DeleteEquipmentPackage: Error deleting package %d: %v", packageID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete package"})
		return
	}

	log.Printf("DeleteEquipmentPackage: Package %d deleted successfully by user %s", packageID, currentUser.Username)
	c.JSON(http.StatusOK, gin.H{
		"message": "Package deleted successfully",
	})
}

// DebugPackageForm shows debug info for package form
func (h *WorkflowHandler) DebugPackageForm(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get available devices for debugging
	availableDevices, err := h.packageRepo.GetAvailableDevices()
	if err != nil {
		log.Printf("DebugPackageForm: Error fetching available devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "equipment_package_form_debug.html", gin.H{
		"title":            "Debug Package Form",
		"user":             currentUser,
		"availableDevices": availableDevices,
	})
}

// ================================================================
// BULK OPERATIONS - PLACEHOLDER METHODS
// ================================================================

// BulkOperationsForm displays the bulk operations interface
func (h *WorkflowHandler) BulkOperationsForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	// TODO: Implement bulk operations interface
	c.HTML(http.StatusOK, "bulk_operations.html", gin.H{
		"title": "Bulk Operations",
		"user":  user,
	})
}

// BulkUpdateDeviceStatus updates multiple device statuses
func (h *WorkflowHandler) BulkUpdateDeviceStatus(c *gin.Context) {
	// TODO: Implement bulk device status update
	log.Printf("BulkUpdateDeviceStatus: Not yet implemented")
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Bulk device status update not yet implemented",
	})
}

// BulkAssignToJob assigns multiple devices to a job
func (h *WorkflowHandler) BulkAssignToJob(c *gin.Context) {
	// TODO: Implement bulk device assignment
	log.Printf("BulkAssignToJob: Not yet implemented")
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Bulk device assignment not yet implemented",
	})
}

// BulkGenerateQRCodes generates QR codes for multiple devices
func (h *WorkflowHandler) BulkGenerateQRCodes(c *gin.Context) {
	// Parse request
	var request struct {
		DeviceIDs    []string `json:"deviceIds" form:"deviceIds"`
		Format       string   `json:"format" form:"format"`       // "pdf" or "zip"
		LabelFormat  string   `json:"labelFormat" form:"labelFormat"` // "simple" or "detailed"
		PrintReady   bool     `json:"printReady" form:"printReady"`
	}

	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate device IDs
	if len(request.DeviceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No device IDs provided"})
		return
	}

	// Default values
	if request.Format == "" {
		request.Format = "pdf"
	}
	if request.LabelFormat == "" {
		request.LabelFormat = "simple"
	}

	log.Printf("Generating QR codes for %d devices, format: %s", len(request.DeviceIDs), request.Format)

	// Fetch device information
	devices := make([]models.Device, 0, len(request.DeviceIDs))
	for _, deviceID := range request.DeviceIDs {
		var device models.Device
		if err := h.db.Preload("Product").Preload("Product.Brand").Where("deviceID = ?", deviceID).First(&device).Error; err != nil {
			log.Printf("Warning: Device %s not found in database, will generate QR anyway", deviceID)
			// Create a minimal device record for QR generation
			device = models.Device{
				DeviceID: deviceID,
				Status:   "unknown",
				Product:  nil, // Will be handled in template
			}
		}
		devices = append(devices, device)
	}

	if request.Format == "zip" {
		// Generate PNG files and create ZIP
		zipBytes, err := h.generateDeviceLabelsZIP(devices, request.LabelFormat, request.PrintReady)
		if err != nil {
			log.Printf("Error generating device labels ZIP: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate device labels ZIP"})
			return
		}

		// Set headers for ZIP download
		filename := fmt.Sprintf("device_labels_%s.zip", time.Now().Format("20060102_150405"))
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

		// Return ZIP
		c.Data(http.StatusOK, "application/zip", zipBytes)
	} else {
		// Generate PDF with multiple labels per page
		pdfBytes, err := h.generateDeviceLabelsPDF(devices, request.LabelFormat, request.PrintReady)
		if err != nil {
			log.Printf("Error generating device labels PDF: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate device labels PDF"})
			return
		}

		// Set headers for PDF download
		filename := fmt.Sprintf("device_labels_%s.pdf", time.Now().Format("20060102_150405"))
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

		// Return PDF
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

// generateDeviceLabelsPDF creates a PDF with multiple device labels per page
func (h *WorkflowHandler) generateDeviceLabelsPDF(devices []models.Device, labelFormat string, printReady bool) ([]byte, error) {
	// Create PDF document - A4 Portrait for multiple labels
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	
	// Load logo if exists
	logoPath := "logo.png"
	logoExists := false
	if _, err := os.Stat(logoPath); err == nil {
		logoExists = true
	}
	
	// Label dimensions - 3x7 grid on A4 (21 labels per page)
	labelWidth := 60.0
	labelHeight := 35.0
	labelsPerRow := 3
	labelsPerCol := 7
	labelsPerPage := labelsPerRow * labelsPerCol
	
	// Process devices in batches per page
	for pageStart := 0; pageStart < len(devices); pageStart += labelsPerPage {
		pdf.AddPage()
		
		// Draw labels for this page
		for i := 0; i < labelsPerPage && pageStart+i < len(devices); i++ {
			device := devices[pageStart+i]
			
			// Calculate position for this label
			row := i / labelsPerRow
			col := i % labelsPerRow
			
			offsetX := 10.0 + float64(col)*labelWidth
			offsetY := 10.0 + float64(row)*labelHeight
			
			h.drawSingleLabel(pdf, device, offsetX, offsetY, labelWidth, labelHeight, logoExists, logoPath)
		}
	}
	
	// Output PDF to bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %v", err)
	}
	
	return buf.Bytes(), nil
}

// drawSingleLabel draws a single device label at the specified position
func (h *WorkflowHandler) drawSingleLabel(pdf *gofpdf.Fpdf, device models.Device, offsetX, offsetY, width, height float64, logoExists bool, logoPath string) {
	// Get product name
	productName := "Unknown Product"
	if device.Product != nil {
		productName = device.Product.Name
	}
	
	// Draw border around label (optional)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Rect(offsetX, offsetY, width, height, "D")
	
	// 1. Logo at right side, vertically centered (if exists)
	if logoExists {
		logoX := offsetX + width - 20
		logoY := offsetY + (height-8)/2
		pdf.Image(logoPath, logoX, logoY, 15, 8, false, "", 0, "")
	}
	
	// Remove the title - start barcode higher up
	
	// 3. Main barcode in center area (moved up since no title)
	barcodeX := offsetX + 2
	barcodeY := offsetY + 4
	barcodeWidth := width - 25 // Leave space for logo
	barcodeHeight := 8.0
	
	// Generate realistic Code128 barcode pattern
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetFillColor(0, 0, 0)
	
	// Use device ID for barcode data
	deviceData := device.DeviceID
	totalBars := len(deviceData) * 8 + 20
	barWidth := barcodeWidth / float64(totalBars)
	
	x := barcodeX
	
	// Start pattern
	for i := 0; i < 3; i++ {
		pdf.Rect(x, barcodeY, barWidth, barcodeHeight, "F")
		x += barWidth * 2
	}
	
	// Data encoding
	for i, char := range deviceData {
		charVal := int(char) + i
		for j := 0; j < 6; j++ {
			if (charVal+j)%3 != 0 {
				pdf.Rect(x, barcodeY, barWidth, barcodeHeight, "F")
			}
			x += barWidth
		}
		x += barWidth
	}
	
	// End pattern
	for i := 0; i < 3; i++ {
		pdf.Rect(x, barcodeY, barWidth, barcodeHeight, "F")
		x += barWidth * 2
	}
	
	// 4. Human readable text under barcode
	pdf.SetXY(barcodeX, barcodeY + barcodeHeight + 1)
	pdf.SetFont("Arial", "", 5)
	pdf.CellFormat(barcodeWidth, 2, device.DeviceID, "", 0, "C", false, 0, "")
	
	// 5. Device information at bottom
	pdf.SetXY(offsetX+2, offsetY+height-10)
	pdf.SetFont("Arial", "B", 7)
	pdf.Cell(0, 3, device.DeviceID)
	
	pdf.SetXY(offsetX+2, offsetY+height-7)
	pdf.SetFont("Arial", "", 6)
	// Truncate product name if too long
	if len(productName) > 25 {
		productName = productName[:22] + "..."
	}
	pdf.Cell(0, 3, productName)
}

// generateDeviceLabelsZIP creates complete label PNG files for each device and packages them in a ZIP
func (h *WorkflowHandler) generateDeviceLabelsZIP(devices []models.Device, labelFormat string, printReady bool) ([]byte, error) {
	// Create ZIP file in memory
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	defer zipWriter.Close()
	
	// Load logo image if exists
	var logoImg image.Image
	logoPath := "logo.png"
	if logoFile, err := os.Open(logoPath); err == nil {
		logoImg, _, _ = image.Decode(logoFile)
		logoFile.Close()
	}
	
	// Create complete label PNG for each device
	for _, device := range devices {
		// Create PNG image for this device
		pngBytes, err := h.createLabelPNG(device, logoImg)
		if err != nil {
			log.Printf("Error generating PNG for device %s: %v", device.DeviceID, err)
			continue
		}
		
		// Create PNG filename
		filename := fmt.Sprintf("label_%s.png", device.DeviceID)
		
		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			log.Printf("Error creating zip file for device %s: %v", device.DeviceID, err)
			continue
		}
		
		_, err = zipFile.Write(pngBytes)
		if err != nil {
			log.Printf("Error writing to zip file for device %s: %v", device.DeviceID, err)
			continue
		}
	}
	
	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %v", err)
	}
	
	return buf.Bytes(), nil
}

// createLabelPNG creates a complete label as PNG image
func (h *WorkflowHandler) createLabelPNG(device models.Device, logoImg image.Image) ([]byte, error) {
	// Label dimensions in pixels (300 DPI equivalent for 100x60mm)
	width := 1200
	height := 700
	
	// Create a new RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	
	// Draw border
	borderColor := color.RGBA{200, 200, 200, 255}
	h.drawRect(img, 10, 10, width-20, height-20, borderColor)
	
	// Generate barcode image
	barcodeBytes, err := h.barcodeService.GenerateDeviceBarcode(device.DeviceID)
	if err == nil {
		if barcodeImg, _, err := image.Decode(bytes.NewReader(barcodeBytes)); err == nil {
			// Scale and position barcode
			barcodeRect := image.Rect(50, 150, 800, 350)
			xdraw.BiLinear.Scale(img, barcodeRect, barcodeImg, barcodeImg.Bounds(), draw.Over, nil)
		}
	}
	
	// Draw logo if available
	if logoImg != nil {
		logoRect := image.Rect(900, 200, 1100, 300)
		xdraw.BiLinear.Scale(img, logoRect, logoImg, logoImg.Bounds(), draw.Over, nil)
	}
	
	// Get product name
	productName := "Unknown Product"
	if device.Product != nil {
		productName = device.Product.Name
		if len(productName) > 30 {
			productName = productName[:27] + "..."
		}
	}
	
	// Draw text
	textColor := color.RGBA{0, 0, 0, 255}
	
	// Device ID (large, bold)
	h.drawText(img, device.DeviceID, 50, 450, 48, textColor)
	
	// Product name (smaller)
	h.drawText(img, productName, 50, 520, 32, textColor)
	
	// Device ID under barcode (small)
	h.drawText(img, device.DeviceID, 350, 380, 24, textColor)
	
	// Convert to PNG bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %v", err)
	}
	
	return buf.Bytes(), nil
}

// drawRect draws a rectangle outline
func (h *WorkflowHandler) drawRect(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	for i := 0; i < width; i++ {
		img.Set(x+i, y, c)
		img.Set(x+i, y+height-1, c)
	}
	for i := 0; i < height; i++ {
		img.Set(x, y+i, c)
		img.Set(x+width-1, y+i, c)
	}
}

// drawText draws text on the image
func (h *WorkflowHandler) drawText(img *image.RGBA, text string, x, y, size int, c color.RGBA) {
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}
	
	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{c},
		Face: basicfont.Face7x13, // Simple built-in font
		Dot:  point,
	}
	
	d.DrawString(text)
}


// ================================================================
// WORKFLOW STATISTICS - PLACEHOLDER METHOD
// ================================================================

// GetWorkflowStats returns workflow statistics
func (h *WorkflowHandler) GetWorkflowStats(c *gin.Context) {
	// TODO: Implement comprehensive workflow statistics
	stats := map[string]interface{}{
		"totalTemplates":     0, // TODO: Get from repository
		"totalPackages":      0, // TODO: Implement packages
		"templatesThisMonth": 0, // TODO: Calculate from database
		"mostUsedTemplate":   nil, // TODO: Get from repository
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}