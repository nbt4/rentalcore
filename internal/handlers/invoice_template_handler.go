package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"

	"go-barcode-webapp/internal/logger"
)

type InvoiceTemplateHandler struct {
	invoiceRepo *repository.InvoiceRepositoryNew
}

func NewInvoiceTemplateHandler(invoiceRepo *repository.InvoiceRepositoryNew) *InvoiceTemplateHandler {
	return &InvoiceTemplateHandler{
		invoiceRepo: invoiceRepo,
	}
}

// ListTemplates displays all invoice templates
func (h *InvoiceTemplateHandler) ListTemplates(c *gin.Context) {
	logger.LogInfo("=== INVOICE TEMPLATE HANDLER CALLED ===")
	logger.LogInfo("ListTemplates: Handler called for path: %s", c.Request.URL.Path)
	logger.LogInfo("ListTemplates: Request method: %s", c.Request.Method)
	logger.LogInfo("ListTemplates: User-Agent: %s", c.Request.Header.Get("User-Agent"))
	user, _ := GetCurrentUser(c)
	logger.LogInfo("ListTemplates: Current user: %+v", user)

	templates, err := h.invoiceRepo.GetAllTemplates()
	if err != nil {
		logger.LogInfo("ListTemplates: Error fetching templates: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load templates",
			"user":  user,
		})
		return
	}

	logger.LogInfo("ListTemplates: Found %d templates", len(templates))
	for i, template := range templates {
		logger.LogInfo("ListTemplates: Template %d: ID=%d, Name=%s, IsActive=%t", i+1, template.TemplateID, template.Name, template.IsActive)
	}

	logger.LogInfo("ListTemplates: Rendering template 'invoice_templates_list.html' with %d templates", len(templates))
	logger.LogInfo("=== ABOUT TO RENDER INVOICE_TEMPLATES_LIST.HTML ===")
	c.HTML(http.StatusOK, "invoice_templates_list.html", gin.H{
		"title":     "Invoice Templates",
		"templates": templates,
		"user":      user,
	})
	logger.LogInfo("=== FINISHED RENDERING INVOICE_TEMPLATES_LIST.HTML ===")
}

// NewTemplateForm displays the template designer for creating a new template
func (h *InvoiceTemplateHandler) NewTemplateForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Use the simplified designer for dummy-proof interface
	c.HTML(http.StatusOK, "invoice_template_designer_simple_new.html", gin.H{
		"title":    "New Invoice Template",
		"user":     user,
		"template": models.InvoiceTemplate{},
		"isEdit":   false,
	})
}

// EditTemplateForm displays the template designer for editing an existing template
func (h *InvoiceTemplateHandler) EditTemplateForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	templateIDStr := c.Param("id")
	templateID64, err := strconv.ParseUint(templateIDStr, 10, 32)
	templateID := uint(templateID64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Invalid template ID",
			"user":  user,
		})
		return
	}

	template, err := h.invoiceRepo.GetTemplateByID(templateID)
	if err != nil {
		logger.LogInfo("EditTemplateForm: Error fetching template: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Template not found",
			"user":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "invoice_template_designer_simple_new.html", gin.H{
		"title":    fmt.Sprintf("Edit Template: %s", template.Name),
		"user":     user,
		"template": template,
		"isEdit":   true,
	})
}

// CreateTemplate creates a new invoice template
func (h *InvoiceTemplateHandler) CreateTemplate(c *gin.Context) {
	logger.LogInfo("CreateTemplate: Handler called")
	user, exists := GetCurrentUser(c)
	if !exists {
		logger.LogInfo("CreateTemplate: User not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	logger.LogInfo("CreateTemplate: User authenticated: %s", user.Username)

	var request struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		HTMLTemplate string `json:"htmlTemplate" binding:"required"`
		CSSStyles    string `json:"cssStyles"`
		IsDefault    bool   `json:"isDefault"`
		IsActive     bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.LogInfo("CreateTemplate: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	template := &models.InvoiceTemplate{
		Name:         request.Name,
		Description:  &request.Description,
		HTMLTemplate: request.HTMLTemplate,
		CSSStyles:    &request.CSSStyles,
		IsDefault:    request.IsDefault,
		IsActive:     request.IsActive,
		CreatedBy:    &user.UserID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	logger.LogInfo("CreateTemplate: Attempting to save template: %s", template.Name)
	err := h.invoiceRepo.CreateTemplate(template)
	if err != nil {
		logger.LogInfo("CreateTemplate: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create template",
			"details": err.Error(),
		})
		return
	}

	logger.LogInfo("CreateTemplate: Template created successfully with ID: %d", template.TemplateID)
	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"message":    "Template created successfully",
		"templateId": template.TemplateID,
	})
}

// UpdateTemplate updates an existing invoice template
func (h *InvoiceTemplateHandler) UpdateTemplate(c *gin.Context) {
	_, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	templateIDStr := c.Param("id")
	templateID64, err := strconv.ParseUint(templateIDStr, 10, 32)
	templateID := uint(templateID64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var request struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		HTMLTemplate string `json:"htmlTemplate" binding:"required"`
		CSSStyles    string `json:"cssStyles"`
		IsDefault    bool   `json:"isDefault"`
		IsActive     bool   `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.LogInfo("UpdateTemplate: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	template := &models.InvoiceTemplate{
		TemplateID:   templateID,
		Name:         request.Name,
		Description:  &request.Description,
		HTMLTemplate: request.HTMLTemplate,
		CSSStyles:    &request.CSSStyles,
		IsDefault:    request.IsDefault,
		IsActive:     request.IsActive,
		UpdatedAt:    time.Now(),
	}

	err = h.invoiceRepo.UpdateTemplate(template)
	if err != nil {
		logger.LogInfo("UpdateTemplate: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update template",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Template updated successfully",
	})
}

// DeleteTemplate deletes an invoice template
func (h *InvoiceTemplateHandler) DeleteTemplate(c *gin.Context) {
	templateIDStr := c.Param("id")
	templateID64, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}
	templateID := uint(templateID64)

	// Check if template is default - don't allow deletion of default templates
	template, err := h.invoiceRepo.GetTemplateByID(templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	if template.IsDefault {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete default template"})
		return
	}

	err = h.invoiceRepo.DeleteTemplate(templateID)
	if err != nil {
		logger.LogInfo("DeleteTemplate: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete template",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// PreviewTemplate shows a preview of the template
func (h *InvoiceTemplateHandler) PreviewTemplate(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	templateIDStr := c.Param("id")
	templateID64, err := strconv.ParseUint(templateIDStr, 10, 32)
	templateID := uint(templateID64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Invalid template ID",
			"user":  user,
		})
		return
	}

	template, err := h.invoiceRepo.GetTemplateByID(templateID)
	if err != nil {
		logger.LogInfo("PreviewTemplate: Error fetching template: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Template not found",
			"user":  user,
		})
		return
	}

	// Get company settings or use placeholder for preview
	company, err := h.invoiceRepo.GetCompanySettings()
	if err != nil {
		logger.LogInfo("PreviewTemplate: Error fetching company settings: %v", err)
		addressLine1 := "[Company Address]"
		postalCode := "[ZIP]"
		city := "[City]"
		phone := "[Phone Number]"
		email := "[Email Address]"
		taxNumber := "[Tax Number]"
		vatNumber := "[VAT Number]"
		
		company = &models.CompanySettings{
			CompanyName:  "[Your Company Name]",
			AddressLine1: &addressLine1,
			PostalCode:   &postalCode,
			City:         &city,
			Phone:        &phone,
			Email:        &email,
			TaxNumber:    &taxNumber,
			VATNumber:    &vatNumber,
		}
	}

	// Create placeholder invoice data for preview
	customerStreet := "[Customer Street]"
	customerHouseNumber := "[No.]"
	customerZIP := "[ZIP]"
	customerCity := "[Customer City]"
	customerFirstName := "[First Name]"
	customerLastName := "[Last Name]"
	customerEmail := "[customer@email.com]"
	customerPhone := "[Phone Number]"
	customerCompanyName := "[Customer Company Name]"
	
	sampleInvoice := &models.Invoice{
		InvoiceNumber: "INV-PREVIEW-001",
		IssueDate:     time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 30),
		Subtotal:      100.00,
		TaxAmount:     19.00,
		TotalAmount:   119.00,
		Customer: &models.Customer{
			CompanyName:  &customerCompanyName,
			Street:       &customerStreet,
			HouseNumber:  &customerHouseNumber,
			ZIP:          &customerZIP,
			City:         &customerCity,
			FirstName:    &customerFirstName,
			LastName:     &customerLastName,
			Email:        &customerEmail,
			PhoneNumber:  &customerPhone,
		},
		LineItems: []models.InvoiceLineItem{
			{
				Description: "[Product/Service Description]",
				Quantity:    1,
				UnitPrice:   100.00,
				TotalPrice:  100.00,
			},
		},
	}

	// Parse CSS styles if they exist
	var designSettings map[string]interface{}
	if template.CSSStyles != nil && *template.CSSStyles != "" {
		if err := json.Unmarshal([]byte(*template.CSSStyles), &designSettings); err != nil {
			logger.LogInfo("PreviewTemplate: Error parsing CSS styles: %v", err)
			designSettings = make(map[string]interface{})
		}
	}

	c.HTML(http.StatusOK, "invoice_preview.html", gin.H{
		"title":          fmt.Sprintf("Preview: %s", template.Name),
		"template":       template,
		"invoice":        sampleInvoice,
		"company":        company,
		"customer":       sampleInvoice.Customer,
		"designSettings": designSettings,
		"user":           user,
	})
}

// GetTemplatesAPI returns templates as JSON for API calls
func (h *InvoiceTemplateHandler) GetTemplatesAPI(c *gin.Context) {
	templates, err := h.invoiceRepo.GetAllTemplates()
	if err != nil {
		logger.LogInfo("GetTemplatesAPI: Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"templates": templates,
	})
}

// SetDefaultTemplate sets a template as the default
func (h *InvoiceTemplateHandler) SetDefaultTemplate(c *gin.Context) {
	templateIDStr := c.Param("id")
	templateID64, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}
	templateID := uint(templateID64)

	err = h.invoiceRepo.SetDefaultTemplate(templateID)
	if err != nil {
		logger.LogInfo("SetDefaultTemplate: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to set default template",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Default template updated successfully",
	})
}