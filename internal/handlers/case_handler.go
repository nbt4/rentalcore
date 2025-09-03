package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type CaseHandler struct {
	caseRepo   *repository.CaseRepository
	deviceRepo *repository.DeviceRepository
}

func NewCaseHandler(caseRepo *repository.CaseRepository, deviceRepo *repository.DeviceRepository) *CaseHandler {
	return &CaseHandler{
		caseRepo:   caseRepo,
		deviceRepo: deviceRepo,
	}
}

// Web interface handlers
func (h *CaseHandler) ListCases(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=400&message=Bad Request&details=%s", err.Error()))
		return
	}

	// DEBUG: Log all query parameters
	fmt.Printf("DEBUG Case Handler: All query params: %+v\n", c.Request.URL.Query())
	
	// Manual parameter extraction to ensure search works
	searchParam := c.Query("search")
	fmt.Printf("DEBUG Case Handler: Raw search parameter: '%s'\n", searchParam)
	if searchParam != "" {
		params.SearchTerm = searchParam
		fmt.Printf("DEBUG Case Handler: Search parameter SET to: '%s'\n", searchParam)
	}
	
	// DEBUG: Log params after binding
	fmt.Printf("DEBUG Case Handler: Final params: SearchTerm='%s'\n", params.SearchTerm)

	cases, err := h.caseRepo.List(params)
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	fmt.Printf("DEBUG: Found %d cases with search term '%s'\n", len(cases), params.SearchTerm)

	SafeHTML(c, http.StatusOK, "cases_list.html", gin.H{
		"title":       "Cases",
		"cases":       cases,
		"params":      params,
		"user":        user,
		"currentPage": "cases",
	})
}



func (h *CaseHandler) NewCaseForm(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/cases")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/cases")
		return
	}

	user, _ := GetCurrentUser(c)
	
	// Get available devices for new case
	availableDevices, err := h.caseRepo.GetAvailableDevices()
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}
	
	c.HTML(http.StatusOK, "case_form.html", gin.H{
		"title": "New Case",
		"case":  &models.Case{},
		"availableDevices": availableDevices,
		"user": user,
	})
}

func (h *CaseHandler) CreateCase(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	status := c.PostForm("status")
	if status == "" {
		status = "free"
	}

	// Parse optional numeric fields
	var weight, width, height, depth *float64
	if weightStr := c.PostForm("weight"); weightStr != "" {
		if w, err := strconv.ParseFloat(weightStr, 64); err == nil {
			weight = &w
		}
	}
	if widthStr := c.PostForm("width"); widthStr != "" {
		if w, err := strconv.ParseFloat(widthStr, 64); err == nil {
			width = &w
		}
	}
	if heightStr := c.PostForm("height"); heightStr != "" {
		if h, err := strconv.ParseFloat(heightStr, 64); err == nil {
			height = &h
		}
	}
	if depthStr := c.PostForm("depth"); depthStr != "" {
		if d, err := strconv.ParseFloat(depthStr, 64); err == nil {
			depth = &d
		}
	}

	case_ := models.Case{
		Name:        name,
		Description: &description,
		Weight:      weight,
		Width:       width,
		Height:      height,
		Depth:       depth,
		Status:      status,
	}

	if err := h.caseRepo.Create(&case_); err != nil {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "case_form.html", gin.H{
			"title": "New Case",
			"case":  &case_,
			"error": err.Error(),
			"user": user,
		})
		return
	}

	c.Redirect(http.StatusFound, "/cases")
}

func (h *CaseHandler) GetCase(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/cases")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/cases")
		return
	}

	user, _ := GetCurrentUser(c)
	
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid case ID", "user": user})
		return
	}

	case_, err := h.caseRepo.GetByID(uint(caseID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Case not found", "user": user})
		return
	}

	c.HTML(http.StatusOK, "case_detail.html", gin.H{
		"case": case_,
		"user": user,
	})
}

func (h *CaseHandler) EditCaseForm(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/cases")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/cases")
		return
	}

	user, _ := GetCurrentUser(c)
	
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid case ID", "user": user})
		return
	}

	case_, err := h.caseRepo.GetByID(uint(caseID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Case not found", "user": user})
		return
	}

	// Get available devices for case management
	availableDevices, err := h.caseRepo.GetAvailableDevicesForCase(uint(caseID))
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	// Debug: Log the number of available devices
	log.Printf("EditCaseForm: Found %d available devices for case %d", len(availableDevices), caseID)
	for i, device := range availableDevices {
		if i < 3 { // Only show first 3 for debugging
			productName := "No Product"
			if device.Product != nil {
				productName = device.Product.Name
			}
			log.Printf("  Device %d: ID='%s', Status='%s', Product='%s'", i+1, device.DeviceID, device.Status, productName)
		}
	}

	c.HTML(http.StatusOK, "case_form.html", gin.H{
		"title": "Edit Case",
		"case":  case_,
		"availableDevices": availableDevices,
		"user": user,
	})
}

func (h *CaseHandler) UpdateCase(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid case ID", "user": user})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	status := c.PostForm("status")

	// Parse optional numeric fields
	var weight, width, height, depth *float64
	if weightStr := c.PostForm("weight"); weightStr != "" {
		if w, err := strconv.ParseFloat(weightStr, 64); err == nil {
			weight = &w
		}
	}
	if widthStr := c.PostForm("width"); widthStr != "" {
		if w, err := strconv.ParseFloat(widthStr, 64); err == nil {
			width = &w
		}
	}
	if heightStr := c.PostForm("height"); heightStr != "" {
		if h, err := strconv.ParseFloat(heightStr, 64); err == nil {
			height = &h
		}
	}
	if depthStr := c.PostForm("depth"); depthStr != "" {
		if d, err := strconv.ParseFloat(depthStr, 64); err == nil {
			depth = &d
		}
	}

	case_ := models.Case{
		CaseID:      uint(caseID),
		Name:        name,
		Description: &description,
		Weight:      weight,
		Width:       width,
		Height:      height,
		Depth:       depth,
		Status:      status,
	}

	// Update the case
	if err := h.caseRepo.Update(&case_); err != nil {
		// Get available devices for error display
		availableDevices, _ := h.deviceRepo.GetAvailableDevicesForCaseManagement()
		c.HTML(http.StatusInternalServerError, "case_form.html", gin.H{
			"title": "Edit Case",
			"case":  &case_,
			"availableDevices": availableDevices,
			"error": err.Error(),
			"user": user,
		})
		return
	}

	// Process device associations
	var deviceIDs []string
	
	// Parse device form data - handle both devices[] and devices[*] formats
	for key, values := range c.Request.PostForm {
		if key == "devices[]" || (strings.HasPrefix(key, "devices[") && strings.HasSuffix(key, "]")) {
			for _, deviceID := range values {
				if deviceID != "" {
					// Avoid duplicates
					found := false
					for _, existing := range deviceIDs {
						if existing == deviceID {
							found = true
							break
						}
					}
					if !found {
						deviceIDs = append(deviceIDs, deviceID)
					}
				}
			}
		}
	}

	// Get current devices in case
	currentDevices, err := h.caseRepo.GetDevicesInCase(uint(caseID))
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	// Create maps for easier lookup
	currentDeviceMap := make(map[string]bool)
	for _, dc := range currentDevices {
		currentDeviceMap[dc.DeviceID] = true
	}

	newDeviceMap := make(map[string]bool)
	for _, deviceID := range deviceIDs {
		newDeviceMap[deviceID] = true
	}

	// Remove devices that are no longer selected
	for _, dc := range currentDevices {
		if !newDeviceMap[dc.DeviceID] {
			if err := h.caseRepo.RemoveDeviceFromCase(uint(caseID), dc.DeviceID); err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
				return
			}
		}
	}

	// Add new devices
	for _, deviceID := range deviceIDs {
		if !currentDeviceMap[deviceID] {
			if err := h.caseRepo.AddDeviceToCase(uint(caseID), deviceID); err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
				return
			}
		}
	}

	c.Redirect(http.StatusFound, "/cases")
}

func (h *CaseHandler) DeleteCase(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	if err := h.caseRepo.Delete(uint(caseID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Case deleted successfully"})
}

// Device mapping handlers
func (h *CaseHandler) CaseDeviceMapping(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid case ID", "user": user})
		return
	}

	case_, err := h.caseRepo.GetByID(uint(caseID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Case not found", "user": user})
		return
	}

	deviceCases, err := h.caseRepo.GetDevicesInCase(uint(caseID))
	if err != nil {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	c.HTML(http.StatusOK, "case_device_mapping.html", gin.H{
		"title":       "Case Device Mapping",
		"case":        case_,
		"deviceCases": deviceCases,
		"user": user,
	})
}

// GetAvailableDevicesWithCaseInfo returns devices with case assignment information for tree view
func (h *CaseHandler) GetAvailableDevicesWithCaseInfo(c *gin.Context) {
	// Get ALL devices (not just available ones) for tree view
	devices, err := h.deviceRepo.GetAvailableDevicesForCaseManagement()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get all device case assignments
	caseAssignments, err := h.caseRepo.GetAllDeviceCaseAssignments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create a map of device ID to case assignment
	deviceToCaseMap := make(map[string]*models.DeviceCase)
	for _, assignment := range caseAssignments {
		deviceToCaseMap[assignment.DeviceID] = &assignment
	}

	// Build tree structure with case assignment info
	treeData, err := h.buildTreeDataWithCaseInfo(devices, deviceToCaseMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"treeData": treeData,
		"devices": devices,
	})
}

func (h *CaseHandler) ScanDeviceToCase(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var request struct {
		DeviceID string `json:"device_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if device exists
	device, err := h.deviceRepo.GetByID(request.DeviceID)
	if err != nil {
		// Try by serial number
		device, err = h.deviceRepo.GetBySerialNo(request.DeviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
	}

	// Check if device is already in a case
	isInCase, err := h.caseRepo.IsDeviceInAnyCase(device.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if isInCase {
		c.JSON(http.StatusConflict, gin.H{"error": "Device is already in a case"})
		return
	}

	// Add device to case
	err = h.caseRepo.AddDeviceToCase(uint(caseID), device.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device added to case successfully",
		"device":  device,
	})
}

func (h *CaseHandler) RemoveDeviceFromCase(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	deviceID := c.Param("deviceId")

	err = h.caseRepo.RemoveDeviceFromCase(uint(caseID), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device removed from case successfully"})
}

// API handlers
func (h *CaseHandler) ListCasesAPI(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cases, err := h.caseRepo.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cases": cases})
}

func (h *CaseHandler) CreateCaseAPI(c *gin.Context) {
	var case_ models.Case
	if err := c.ShouldBindJSON(&case_); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.caseRepo.Create(&case_); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, case_)
}

func (h *CaseHandler) GetCaseAPI(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	case_, err := h.caseRepo.GetByID(uint(caseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"case": case_})
}

func (h *CaseHandler) UpdateCaseAPI(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	var case_ models.Case
	if err := c.ShouldBindJSON(&case_); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	case_.CaseID = uint(caseID)
	if err := h.caseRepo.Update(&case_); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, case_)
}

func (h *CaseHandler) DeleteCaseAPI(c *gin.Context) {
	caseIDStr := c.Param("id")
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	if err := h.caseRepo.Delete(uint(caseID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Case deleted successfully"})
}

// GetCaseDevicesAPI returns devices in a case as JSON
func (h *CaseHandler) GetCaseDevicesAPI(c *gin.Context) {
	caseIDStr := c.Param("id")
	log.Printf("GetCaseDevicesAPI: Getting devices for case ID: %s", caseIDStr)
	
	caseID, err := strconv.ParseUint(caseIDStr, 10, 32)
	if err != nil {
		log.Printf("GetCaseDevicesAPI: Invalid case ID: %s, error: %v", caseIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		return
	}

	deviceCases, err := h.caseRepo.GetDevicesInCase(uint(caseID))
	if err != nil {
		log.Printf("GetCaseDevicesAPI: Database error for case %d: %v", caseID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure we always return an array, never null
	if deviceCases == nil {
		deviceCases = []models.DeviceCase{}
	}

	// Extract the actual devices from the DeviceCase relations
	devices := make([]models.Device, len(deviceCases))
	for i, deviceCase := range deviceCases {
		devices[i] = deviceCase.Device
	}

	log.Printf("GetCaseDevicesAPI: Found %d devices for case %d", len(devices), caseID)
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// Tree structure definitions for case device management
type CaseTreeCategory struct {
	ID            uint                   `json:"id"`
	Name          string                 `json:"name"`
	DeviceCount   int                    `json:"device_count"`
	DirectDevices []CaseTreeDevice       `json:"direct_devices"`
	Subcategories []CaseTreeSubcategory  `json:"subcategories"`
}

type CaseTreeSubcategory struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	DeviceCount       int                         `json:"device_count"`
	DirectDevices     []CaseTreeDevice            `json:"direct_devices"`
	Subbiercategories []CaseTreeSubbiercategory   `json:"subbiercategories"`
}

type CaseTreeSubbiercategory struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	DeviceCount int              `json:"device_count"`
	Devices     []CaseTreeDevice `json:"devices"`
}

type CaseTreeDevice struct {
	DeviceID       string  `json:"deviceID"`
	ProductName    string  `json:"productName"`
	SerialNumber   string  `json:"serialNumber"`
	Status         string  `json:"status"`
	Available      bool    `json:"available"`
	AssignedToCase *uint   `json:"assignedToCase,omitempty"`
	CaseName       string  `json:"caseName,omitempty"`
}

// buildTreeDataWithCaseInfo creates tree structure with case assignment information
func (h *CaseHandler) buildTreeDataWithCaseInfo(devices []models.Device, deviceToCaseMap map[string]*models.DeviceCase) ([]CaseTreeCategory, error) {
	if len(devices) == 0 {
		return []CaseTreeCategory{}, nil
	}

	// Group devices by category structure
	categoryMap := make(map[uint]*CaseTreeCategory)
	
	for _, device := range devices {
		if device.Product == nil {
			continue // Skip devices without products
		}

		categoryID := device.Product.CategoryID
		if categoryID == nil {
			continue // Skip devices without category
		}

		// Get or create category
		var category *CaseTreeCategory
		if existingCategory, exists := categoryMap[*categoryID]; exists {
			category = existingCategory
		} else {
			categoryName := "Unknown Category"
			if device.Product.Category != nil {
				categoryName = device.Product.Category.Name
			}
			
			category = &CaseTreeCategory{
				ID:            *categoryID,
				Name:          categoryName,
				DeviceCount:   0,
				DirectDevices: []CaseTreeDevice{},
				Subcategories: make([]CaseTreeSubcategory, 0),
			}
			categoryMap[*categoryID] = category
		}

		// Convert device to tree device with case info
		treeDevice := h.convertToTreeDeviceWithCaseInfo(device, deviceToCaseMap)

		// Check if device has subcategory
		if device.Product.SubcategoryID != nil {
			subcategoryID := *device.Product.SubcategoryID
			
			// Find or create subcategory
			var subcategory *CaseTreeSubcategory
			for i := range category.Subcategories {
				if category.Subcategories[i].ID == subcategoryID {
					subcategory = &category.Subcategories[i]
					break
				}
			}
			
			if subcategory == nil {
				subcategoryName := "Unknown Subcategory"
				if device.Product.Subcategory != nil {
					subcategoryName = device.Product.Subcategory.Name
				}
				
				newSubcategory := CaseTreeSubcategory{
					ID:                subcategoryID,
					Name:              subcategoryName,
					DeviceCount:       0,
					DirectDevices:     []CaseTreeDevice{},
					Subbiercategories: make([]CaseTreeSubbiercategory, 0),
				}
				category.Subcategories = append(category.Subcategories, newSubcategory)
				subcategory = &category.Subcategories[len(category.Subcategories)-1]
			}

			// Check if device has subbiercategory
			if device.Product.SubbiercategoryID != nil {
				subbiercategoryID := *device.Product.SubbiercategoryID
				
				// Find or create subbiercategory
				var subbiercategory *CaseTreeSubbiercategory
				for i := range subcategory.Subbiercategories {
					if subcategory.Subbiercategories[i].ID == subbiercategoryID {
						subbiercategory = &subcategory.Subbiercategories[i]
						break
					}
				}
				
				if subbiercategory == nil {
					subbiercategoryName := "Unknown Subbiercategory"
					if device.Product.Subbiercategory != nil {
						subbiercategoryName = device.Product.Subbiercategory.Name
					}
					
					newSubbiercategory := CaseTreeSubbiercategory{
						ID:          subbiercategoryID,
						Name:        subbiercategoryName,
						DeviceCount: 0,
						Devices:     []CaseTreeDevice{},
					}
					subcategory.Subbiercategories = append(subcategory.Subbiercategories, newSubbiercategory)
					subbiercategory = &subcategory.Subbiercategories[len(subcategory.Subbiercategories)-1]
				}
				
				// Add device to subbiercategory - count in all levels
				subbiercategory.Devices = append(subbiercategory.Devices, treeDevice)
				subbiercategory.DeviceCount++
				subcategory.DeviceCount++
				category.DeviceCount++
			} else {
				// Add device directly to subcategory - count in subcategory and category
				subcategory.DirectDevices = append(subcategory.DirectDevices, treeDevice)
				subcategory.DeviceCount++
				category.DeviceCount++
			}
		} else {
			// Add device directly to category - count only in category
			category.DirectDevices = append(category.DirectDevices, treeDevice)
			category.DeviceCount++
		}
	}

	// Convert map to slice
	var treeCategories []CaseTreeCategory
	for _, category := range categoryMap {
		treeCategories = append(treeCategories, *category)
	}

	return treeCategories, nil
}

// convertToTreeDeviceWithCaseInfo converts a Device to CaseTreeDevice with case assignment info
func (h *CaseHandler) convertToTreeDeviceWithCaseInfo(device models.Device, deviceToCaseMap map[string]*models.DeviceCase) CaseTreeDevice {
	productName := "Unknown Product"
	if device.Product != nil {
		productName = device.Product.Name
	}

	serialNumber := ""
	if device.SerialNumber != nil {
		serialNumber = *device.SerialNumber
	}

	treeDevice := CaseTreeDevice{
		DeviceID:     device.DeviceID,
		ProductName:  productName,
		SerialNumber: serialNumber,
		Status:       device.Status,
		Available:    true, // Default to available
	}

	// Check if device is assigned to a case
	if caseAssignment, exists := deviceToCaseMap[device.DeviceID]; exists {
		treeDevice.Available = false
		treeDevice.AssignedToCase = &caseAssignment.CaseID
		
		// Try to get case name if we have case information
		if caseAssignment.Case.Name != "" {
			treeDevice.CaseName = caseAssignment.Case.Name
		}
	}

	return treeDevice
}