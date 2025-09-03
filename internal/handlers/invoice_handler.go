package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
)

type InvoiceHandlerNew struct {
	invoiceRepo  *repository.InvoiceRepositoryNew
	customerRepo *repository.CustomerRepository
	jobRepo      *repository.JobRepository
	deviceRepo   *repository.DeviceRepository
	packageRepo  *repository.EquipmentPackageRepository
	productRepo  *repository.ProductRepository
	pdfService   *services.PDFServiceNew
}

func NewInvoiceHandlerNew(
	invoiceRepo *repository.InvoiceRepositoryNew,
	customerRepo *repository.CustomerRepository,
	jobRepo *repository.JobRepository,
	deviceRepo *repository.DeviceRepository,
	packageRepo *repository.EquipmentPackageRepository,
	productRepo *repository.ProductRepository,
	pdfConfig *config.PDFConfig,
) *InvoiceHandlerNew {
	return &InvoiceHandlerNew{
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		jobRepo:      jobRepo,
		deviceRepo:   deviceRepo,
		packageRepo:  packageRepo,
		productRepo:  productRepo,
		pdfService:   services.NewPDFServiceNew(pdfConfig),
	}
}

// CreateInvoice creates a new invoice
func (h *InvoiceHandlerNew) CreateInvoice(c *gin.Context) {
	_, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var request models.InvoiceCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("CreateInvoice: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Additional validation
	if err := request.Validate(); err != nil {
		log.Printf("CreateInvoice: Business validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Create invoice
	invoice, err := h.invoiceRepo.CreateInvoice(&request)
	if err != nil {
		log.Printf("CreateInvoice: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create invoice",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"message":       "Invoice created successfully",
		"invoiceId":     invoice.InvoiceID,
		"invoiceNumber": invoice.InvoiceNumber,
	})
}

// GenerateInvoicePDF generates and downloads a PDF for an invoice
func (h *InvoiceHandlerNew) GenerateInvoicePDF(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invoice ID"})
		return
	}

	// Get invoice
	invoice, err := h.invoiceRepo.GetInvoiceByID(invoiceID)
	if err != nil {
		log.Printf("GenerateInvoicePDF: Error fetching invoice: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}

	// Get company settings
	company, err := h.invoiceRepo.GetCompanySettings()
	if err != nil {
		log.Printf("GenerateInvoicePDF: Error fetching company settings: %v", err)
		company = &models.CompanySettings{CompanyName: "RentalCore Company"}
	}

	// Get invoice settings
	settings, err := h.invoiceRepo.GetAllInvoiceSettings()
	if err != nil {
		log.Printf("GenerateInvoicePDF: Error fetching settings: %v", err)
		settings = &models.InvoiceSettings{CurrencySymbol: "€"}
	}

	// Generate PDF
	pdfBytes, err := h.pdfService.GenerateInvoicePDF(invoice, company, settings)
	if err != nil {
		log.Printf("GenerateInvoicePDF: Error generating PDF: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate PDF",
			"details": err.Error(),
		})
		return
	}

	// Validate PDF content
	if len(pdfBytes) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generated PDF is empty"})
		return
	}

	// Validate PDF content - ensure it's actually a PDF, not HTML
	if len(pdfBytes) < 4 || string(pdfBytes[:4]) != "%PDF" {
		log.Printf("GenerateInvoicePDF: Invalid PDF content returned (not starting with %%PDF)")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "PDF generation failed - invalid PDF format",
			"details": "The generated content is not a valid PDF file",
		})
		return
	}

	// Set headers for PDF download
	filename := fmt.Sprintf("Invoice_%s.pdf", strings.ReplaceAll(invoice.InvoiceNumber, "/", "_"))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", strconv.Itoa(len(pdfBytes)))

	// Send PDF
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// GetInvoicesAPI returns invoices as JSON
func (h *InvoiceHandlerNew) GetInvoicesAPI(c *gin.Context) {
	var filter models.InvoiceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoices, totalCount, err := h.invoiceRepo.GetInvoices(&filter)
	if err != nil {
		log.Printf("GetInvoicesAPI: Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load invoices"})
		return
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"invoices":   invoices,
		"totalCount": totalCount,
		"totalPages": totalPages,
		"filter":     filter,
	})
}

// GetInvoiceStatsAPI returns invoice statistics
func (h *InvoiceHandlerNew) GetInvoiceStatsAPI(c *gin.Context) {
	stats, err := h.invoiceRepo.GetInvoiceStats()
	if err != nil {
		log.Printf("GetInvoiceStatsAPI: Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get invoice statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

// ================================================================
// WEB INTERFACE METHODS
// ================================================================

// ListInvoices displays all invoices
func (h *InvoiceHandlerNew) ListInvoices(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	// Parse filter parameters
	var filter models.InvoiceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		log.Printf("ListInvoices: Filter binding error: %v", err)
	}

	// Set default pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Get invoices using new repository
	invoices, _, err := h.invoiceRepo.GetInvoices(&filter)
	if err != nil {
		log.Printf("ListInvoices: Error fetching invoices: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load invoices",
			"user":  user,
		})
		return
	}

	SafeHTML(c, http.StatusOK, "invoices_list.html", gin.H{
		"title":       "Invoices",
		"invoices":    invoices,
		"user":        user,
		"currentPage": "invoices",
	})
}

// NewInvoiceForm displays the form for creating a new invoice
func (h *InvoiceHandlerNew) NewInvoiceForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get customers for dropdown
	customers, err := h.customerRepo.List(&models.FilterParams{Limit: 1000})
	if err != nil {
		log.Printf("NewInvoiceForm: Error fetching customers: %v", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load customers",
			"user":  user,
		})
		return
	}

	// Get jobs for dropdown
	jobs, err := h.jobRepo.List(&models.FilterParams{Limit: 1000})
	if err != nil {
		log.Printf("NewInvoiceForm: Error fetching jobs: %v", err)
		jobs = []models.JobWithDetails{} // Continue with empty jobs list
	}

	// Get products for dropdown
	products, err := h.productRepo.List(&models.FilterParams{Limit: 1000})
	if err != nil {
		log.Printf("NewInvoiceForm: Error fetching products: %v", err)
		products = []models.Product{} // Continue with empty products list
	}

	// Generate a preview invoice number
	previewInvoiceNumber, err := h.invoiceRepo.GeneratePreviewInvoiceNumber()
	if err != nil {
		log.Printf("NewInvoiceForm: Error generating preview invoice number: %v", err)
		previewInvoiceNumber = "INV-PREVIEW" // Fallback
	}

	c.HTML(http.StatusOK, "invoice_form_new.html", gin.H{
		"title":            "New Invoice",
		"user":             user,
		"customers":        customers,
		"jobs":             jobs,
		"products":         products,
		"action":           "create",
		"defaultIssueDate": time.Now().Format("2006-01-02"),
		"defaultDueDate":   time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		"previewInvoiceNumber": previewInvoiceNumber,
	})
}

// GetInvoice displays a single invoice
func (h *InvoiceHandlerNew) GetInvoice(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Invalid invoice ID",
			"user":  user,
		})
		return
	}

	// Get invoice using new repository
	invoice, err := h.invoiceRepo.GetInvoiceByID(invoiceID)
	if err != nil {
		log.Printf("GetInvoice: Error fetching invoice: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Invoice not found",
			"user":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "invoice_detail.html", gin.H{
		"title":   fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
		"invoice": invoice,
		"user":    user,
	})
}

// EditInvoiceForm displays the form for editing an invoice
func (h *InvoiceHandlerNew) EditInvoiceForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Invalid invoice ID",
			"user":  user,
		})
		return
	}

	// Get invoice
	invoice, err := h.invoiceRepo.GetInvoiceByID(invoiceID)
	if err != nil {
		log.Printf("EditInvoiceForm: Error fetching invoice: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Invoice not found",
			"user":  user,
		})
		return
	}

	// Get customers for dropdown
	customers, err := h.customerRepo.List(&models.FilterParams{Limit: 1000})
	if err != nil {
		log.Printf("EditInvoiceForm: Error fetching customers: %v", err)
		customers = []models.Customer{}
	}

	// Get jobs for dropdown
	jobs, err := h.jobRepo.List(&models.FilterParams{Limit: 1000})
	if err != nil {
		log.Printf("EditInvoiceForm: Error fetching jobs: %v", err)
		jobs = []models.JobWithDetails{}
	}

	c.HTML(http.StatusOK, "invoice_form_new.html", gin.H{
		"title":     fmt.Sprintf("Edit Invoice %s", invoice.InvoiceNumber),
		"user":      user,
		"invoice":   invoice,
		"customers": customers,
		"jobs":      jobs,
		"action":    "edit",
	})
}

// PreviewInvoice shows a preview of the invoice
func (h *InvoiceHandlerNew) PreviewInvoice(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Invalid invoice ID",
			"user":  user,
		})
		return
	}

	// Get invoice
	invoice, err := h.invoiceRepo.GetInvoiceByID(invoiceID)
	if err != nil {
		log.Printf("PreviewInvoice: Error fetching invoice: %v", err)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Invoice not found",
			"user":  user,
		})
		return
	}

	// Get company settings
	company, err := h.invoiceRepo.GetCompanySettings()
	if err != nil {
		log.Printf("PreviewInvoice: Error fetching company settings: %v", err)
		company = &models.CompanySettings{CompanyName: "RentalCore Company"}
	}

	// Get invoice settings
	settings, err := h.invoiceRepo.GetAllInvoiceSettings()
	if err != nil {
		log.Printf("PreviewInvoice: Error fetching settings: %v", err)
		settings = &models.InvoiceSettings{CurrencySymbol: "€"}
	}

	c.HTML(http.StatusOK, "invoice_preview_new.html", gin.H{
		"title":    fmt.Sprintf("Preview Invoice %s", invoice.InvoiceNumber),
		"invoice":  invoice,
		"company":  company,
		"settings": settings,
		"user":     user,
	})
}

// UpdateInvoice updates an existing invoice
func (h *InvoiceHandlerNew) UpdateInvoice(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invoice ID"})
		return
	}

	var request models.InvoiceCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("UpdateInvoice: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Additional validation
	if err := request.Validate(); err != nil {
		log.Printf("UpdateInvoice: Business validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Update invoice using new repository
	invoice, err := h.invoiceRepo.UpdateInvoice(invoiceID, &request)
	if err != nil {
		log.Printf("UpdateInvoice: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update invoice",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Invoice updated successfully",
		"invoiceId":     invoice.InvoiceID,
		"invoiceNumber": invoice.InvoiceNumber,
	})
}

// DeleteInvoice deletes an invoice
func (h *InvoiceHandlerNew) DeleteInvoice(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invoice ID"})
		return
	}

	// Delete invoice using new repository
	err = h.invoiceRepo.DeleteInvoice(invoiceID)
	if err != nil {
		log.Printf("DeleteInvoice: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete invoice",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invoice deleted successfully",
	})
}

// GetProductDetails returns product details including price
func (h *InvoiceHandlerNew) GetProductDetails(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.productRepo.GetByID(uint(productID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Get devices for this product
	devices, err := h.deviceRepo.GetByProductID(uint(productID))
	if err != nil {
		log.Printf("GetProductDetails: Error fetching devices: %v", err)
		devices = []models.Device{} // Continue with empty devices list
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product,
		"devices": devices,
	})
}

// GetDevicesByProduct returns all devices for a specific product
func (h *InvoiceHandlerNew) GetDevicesByProduct(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	devices, err := h.deviceRepo.GetByProductID(uint(productID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
	})
}

// UpdateInvoiceStatus updates the status of an invoice
func (h *InvoiceHandlerNew) UpdateInvoiceStatus(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invoice ID"})
		return
	}

	var request struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Update status using new repository
	err = h.invoiceRepo.UpdateInvoiceStatus(invoiceID, request.Status)
	if err != nil {
		log.Printf("UpdateInvoiceStatus: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update invoice status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invoice status updated successfully",
	})
}

