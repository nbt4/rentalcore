package repository

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type InvoiceRepositoryNew struct {
	db *Database
}

func NewInvoiceRepositoryNew(db *Database) *InvoiceRepositoryNew {
	return &InvoiceRepositoryNew{db: db}
}

// GetDB returns the database instance for direct queries
func (r *InvoiceRepositoryNew) GetDB() *gorm.DB {
	return r.db.DB
}

// ================================================================
// CORE INVOICE OPERATIONS
// ================================================================

// CreateInvoice creates a new invoice with proper validation
func (r *InvoiceRepositoryNew) CreateInvoice(request *models.InvoiceCreateRequest) (*models.Invoice, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}

	var invoice *models.Invoice
	err := r.db.DB.Transaction(func(tx *gorm.DB) error {
		// Generate invoice number
		invoiceNumber, err := r.generateInvoiceNumber(tx)
		if err != nil {
			return fmt.Errorf("failed to generate invoice number: %v", err)
		}

		// Create invoice
		invoice = &models.Invoice{
			InvoiceNumber:   invoiceNumber,
			CustomerID:      request.CustomerID,
			JobID:           request.JobID,
			TemplateID:      request.TemplateID,
			Status:          "draft",
			IssueDate:       request.IssueDate,
			DueDate:         request.DueDate,
			PaymentTerms:    request.PaymentTerms,
			TaxRate:         request.TaxRate,
			DiscountAmount:  request.DiscountAmount,
			Notes:           request.Notes,
			TermsConditions: request.TermsConditions,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Create line items
		for i, itemRequest := range request.LineItems {
			lineItem := models.InvoiceLineItem{
				ItemType:        itemRequest.ItemType,
				DeviceID:        itemRequest.DeviceID,
				PackageID:       itemRequest.PackageID,
				Description:     itemRequest.Description,
				Quantity:        itemRequest.Quantity,
				UnitPrice:       itemRequest.UnitPrice,
				RentalStartDate: itemRequest.RentalStartDate,
				RentalEndDate:   itemRequest.RentalEndDate,
				RentalDays:      itemRequest.RentalDays,
				SortOrder:       func() *uint { order := uint(i); return &order }(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			lineItem.CalculateTotal()
			invoice.LineItems = append(invoice.LineItems, lineItem)
		}

		// Calculate totals
		invoice.CalculateTotals()

		// Save to database
		if err := tx.Create(invoice).Error; err != nil {
			return fmt.Errorf("failed to create invoice: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Load relationships for return
	if err := r.db.DB.Preload("Customer").
		Preload("Job").
		Preload("Template").
		Preload("LineItems").
		First(invoice, invoice.InvoiceID).Error; err != nil {
		return nil, fmt.Errorf("failed to load created invoice: %v", err)
	}

	log.Printf("Successfully created invoice %s with ID %d, CustomerID: %d", invoice.InvoiceNumber, invoice.InvoiceID, invoice.CustomerID)
	if invoice.Customer != nil {
		log.Printf("Loaded customer: ID=%d, Name=%s", invoice.Customer.CustomerID, invoice.Customer.GetDisplayName())
	} else {
		log.Printf("WARNING: Customer not loaded for CustomerID %d", invoice.CustomerID)
	}
	return invoice, nil
}

// GetInvoiceByID retrieves an invoice by ID with all relationships
func (r *InvoiceRepositoryNew) GetInvoiceByID(id uint64) (*models.Invoice, error) {
	var invoice models.Invoice

	if err := r.db.DB.
		Preload("Customer").
		Preload("Job").
		Preload("Template").
		Preload("LineItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, line_item_id ASC")
		}).
		Preload("LineItems.Device").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payment_date DESC")
		}).
		First(&invoice, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invoice with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get invoice: %v", err)
	}

	return &invoice, nil
}

// GetInvoiceByNumber retrieves an invoice by invoice number
func (r *InvoiceRepositoryNew) GetInvoiceByNumber(invoiceNumber string) (*models.Invoice, error) {
	var invoice models.Invoice

	if err := r.db.DB.
		Preload("Customer").
		Preload("Job").
		Preload("Template").
		Preload("LineItems", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, line_item_id ASC")
		}).
		Preload("LineItems.Device").
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payment_date DESC")
		}).
		Where("invoice_number = ?", invoiceNumber).
		First(&invoice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invoice %s not found", invoiceNumber)
		}
		return nil, fmt.Errorf("failed to get invoice: %v", err)
	}

	return &invoice, nil
}

// UpdateInvoice updates an existing invoice
func (r *InvoiceRepositoryNew) UpdateInvoice(id uint64, request *models.InvoiceCreateRequest) (*models.Invoice, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}

	var invoice models.Invoice
	err := r.db.DB.Transaction(func(tx *gorm.DB) error {
		// Get existing invoice
		if err := tx.First(&invoice, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("invoice with ID %d not found", id)
			}
			return fmt.Errorf("failed to get invoice: %v", err)
		}

		// Only allow editing draft invoices
		if invoice.Status != "draft" {
			return fmt.Errorf("only draft invoices can be edited")
		}

		// Update invoice fields
		invoice.CustomerID = request.CustomerID
		invoice.JobID = request.JobID
		invoice.TemplateID = request.TemplateID
		invoice.IssueDate = request.IssueDate
		invoice.DueDate = request.DueDate
		invoice.PaymentTerms = request.PaymentTerms
		invoice.TaxRate = request.TaxRate
		invoice.DiscountAmount = request.DiscountAmount
		invoice.Notes = request.Notes
		invoice.TermsConditions = request.TermsConditions
		invoice.UpdatedAt = time.Now()

		// Delete existing line items
		if err := tx.Where("invoice_id = ?", id).Delete(&models.InvoiceLineItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing line items: %v", err)
		}

		// Create new line items
		invoice.LineItems = []models.InvoiceLineItem{}
		for i, itemRequest := range request.LineItems {
			lineItem := models.InvoiceLineItem{
				InvoiceID:       invoice.InvoiceID,
				ItemType:        itemRequest.ItemType,
				DeviceID:        itemRequest.DeviceID,
				PackageID:       itemRequest.PackageID,
				Description:     itemRequest.Description,
				Quantity:        itemRequest.Quantity,
				UnitPrice:       itemRequest.UnitPrice,
				RentalStartDate: itemRequest.RentalStartDate,
				RentalEndDate:   itemRequest.RentalEndDate,
				RentalDays:      itemRequest.RentalDays,
				SortOrder:       func() *uint { order := uint(i); return &order }(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			lineItem.CalculateTotal()
			invoice.LineItems = append(invoice.LineItems, lineItem)
		}

		// Calculate totals
		invoice.CalculateTotals()

		// Save invoice
		if err := tx.Save(&invoice).Error; err != nil {
			return fmt.Errorf("failed to update invoice: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Load relationships for return
	if err := r.db.DB.Preload("Customer").
		Preload("Job").
		Preload("Template").
		Preload("LineItems").
		First(&invoice, invoice.InvoiceID).Error; err != nil {
		return nil, fmt.Errorf("failed to load updated invoice: %v", err)
	}

	log.Printf("Successfully updated invoice %s", invoice.InvoiceNumber)
	return &invoice, nil
}

// UpdateInvoiceStatus updates the status of an invoice
func (r *InvoiceRepositoryNew) UpdateInvoiceStatus(id uint64, status string) error {
	// Validate status
	validStatuses := []string{"draft", "sent", "paid", "overdue", "cancelled"}
	isValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid status: %s", status)
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Set special timestamps based on status
	now := time.Now()
	switch status {
	case "sent":
		updates["sent_at"] = &now
	case "paid":
		updates["paid_at"] = &now
	}

	result := r.db.DB.Model(&models.Invoice{}).
		Where("invoice_id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update invoice status: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invoice with ID %d not found", id)
	}

	log.Printf("Successfully updated invoice %d status to %s", id, status)
	return nil
}

// DeleteInvoice soft deletes an invoice by setting status to cancelled
func (r *InvoiceRepositoryNew) DeleteInvoice(id uint64) error {
	result := r.db.DB.Model(&models.Invoice{}).
		Where("invoice_id = ?", id).
		Updates(map[string]interface{}{
			"status":     "cancelled",
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to delete invoice: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invoice with ID %d not found", id)
	}

	log.Printf("Successfully deleted invoice %d", id)
	return nil
}

// ================================================================
// INVOICE NUMBER GENERATION
// ================================================================

// generateInvoiceNumber generates a unique invoice number
func (r *InvoiceRepositoryNew) generateInvoiceNumber(tx *gorm.DB) (string, error) {
	// Get settings
	prefix := r.getSettingWithDefault("invoice_number_prefix", "RE")
	format := r.getSettingWithDefault("invoice_number_format", "{prefix}{sequence:4}")

	// Get current year
	year := time.Now().Year()

	// Find the highest existing number for this prefix
	var maxNumber int
	pattern := prefix + "%"
	
	err := tx.Raw(`
		SELECT COALESCE(MAX(
			CAST(
				SUBSTRING(invoice_number FROM ? FOR 10) AS UNSIGNED
			)
		), 1000) as max_num
		FROM invoices 
		WHERE invoice_number LIKE ?
	`, len(prefix)+1, pattern).Scan(&maxNumber).Error
	
	if err != nil {
		// Fallback: use timestamp-based number
		maxNumber = int(time.Now().Unix()) % 100000
		log.Printf("Warning: Could not get max invoice number, using fallback: %d", maxNumber)
	}

	nextNumber := maxNumber + 1

	// Generate invoice number based on format
	invoiceNumber := strings.ReplaceAll(format, "{prefix}", prefix)
	invoiceNumber = strings.ReplaceAll(invoiceNumber, "{year}", fmt.Sprintf("%d", year))
	invoiceNumber = strings.ReplaceAll(invoiceNumber, "{sequence:4}", fmt.Sprintf("%04d", nextNumber))

	// Ensure uniqueness
	var count int64
	for i := 0; i < 10; i++ { // Max 10 attempts
		err = tx.Model(&models.Invoice{}).Where("invoice_number = ?", invoiceNumber).Count(&count).Error
		if err != nil {
			return "", fmt.Errorf("failed to check invoice number uniqueness: %v", err)
		}
		if count == 0 {
			break
		}
		// If number exists, increment and try again
		nextNumber++
		invoiceNumber = strings.ReplaceAll(format, "{prefix}", prefix)
		invoiceNumber = strings.ReplaceAll(invoiceNumber, "{year}", fmt.Sprintf("%d", year))
		invoiceNumber = strings.ReplaceAll(invoiceNumber, "{sequence:4}", fmt.Sprintf("%04d", nextNumber))
	}

	if count > 0 {
		return "", fmt.Errorf("failed to generate unique invoice number after 10 attempts")
	}

	return invoiceNumber, nil
}

// GeneratePreviewInvoiceNumber generates a preview invoice number for the form
func (r *InvoiceRepositoryNew) GeneratePreviewInvoiceNumber() (string, error) {
	// Get settings from config - using the configured format
	prefix := "INV-"
	format := "{prefix}{year}{month}{sequence:4}"
	
	// Get current year and month
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	
	// Find the highest existing number for this prefix and year/month
	var maxNumber int
	pattern := prefix + year + month + "%"
	
	err := r.db.DB.Raw(`
		SELECT COALESCE(MAX(
			CAST(
				SUBSTRING(invoice_number FROM ? FOR 4) AS UNSIGNED
			)
		), 0) as max_num
		FROM invoices 
		WHERE invoice_number LIKE ?
	`, len(prefix)+len(year)+len(month)+1, pattern).Scan(&maxNumber).Error
	
	if err != nil {
		// Fallback: use 1 as the next number
		maxNumber = 0
		log.Printf("Warning: Could not get max invoice number for preview, using fallback")
	}
	
	nextNumber := maxNumber + 1
	
	// Generate invoice number based on format
	invoiceNumber := strings.ReplaceAll(format, "{prefix}", prefix)
	invoiceNumber = strings.ReplaceAll(invoiceNumber, "{year}", year)
	invoiceNumber = strings.ReplaceAll(invoiceNumber, "{month}", month)
	invoiceNumber = strings.ReplaceAll(invoiceNumber, "{sequence:4}", fmt.Sprintf("%04d", nextNumber))
	
	return invoiceNumber, nil
}

// ================================================================
// LIST AND FILTER OPERATIONS
// ================================================================

// GetInvoices returns a paginated list of invoices with filters
func (r *InvoiceRepositoryNew) GetInvoices(filter *models.InvoiceFilter) ([]models.Invoice, int64, error) {
	var invoices []models.Invoice
	var totalCount int64

	query := r.db.DB.Model(&models.Invoice{}).
		Preload("Customer").
		Preload("Job").
		Preload("Template")

	// Apply filters
	if filter != nil {
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.CustomerID != nil {
			query = query.Where("customer_id = ?", *filter.CustomerID)
		}
		if filter.JobID != nil {
			query = query.Where("job_id = ?", *filter.JobID)
		}
		if filter.StartDate != nil {
			query = query.Where("issue_date >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("issue_date <= ?", *filter.EndDate)
		}
		if filter.MinAmount != nil {
			query = query.Where("total_amount >= ?", *filter.MinAmount)
		}
		if filter.MaxAmount != nil {
			query = query.Where("total_amount <= ?", *filter.MaxAmount)
		}
		if filter.OverdueOnly {
			query = query.Where("due_date < ? AND status NOT IN ('paid', 'cancelled')", time.Now())
		}
		if filter.SearchTerm != "" {
			searchTerm := "%" + filter.SearchTerm + "%"
			query = query.Where("invoice_number LIKE ? OR notes LIKE ?", searchTerm, searchTerm)
		}
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %v", err)
	}

	// Apply pagination and order
	if filter != nil {
		if filter.PageSize > 0 {
			query = query.Limit(filter.PageSize)
		}
		if filter.Page > 0 {
			offset := (filter.Page - 1) * filter.PageSize
			query = query.Offset(offset)
		}
	}

	query = query.Order("issue_date DESC, created_at DESC")

	if err := query.Find(&invoices).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get invoices: %v", err)
	}

	return invoices, totalCount, nil
}

// ================================================================
// TEMPLATE OPERATIONS
// ================================================================

// GetAllTemplates returns all invoice templates
func (r *InvoiceRepositoryNew) GetAllTemplates() ([]models.InvoiceTemplate, error) {
	var templates []models.InvoiceTemplate

	if err := r.db.DB.Order("is_default DESC, name ASC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to get templates: %v", err)
	}

	return templates, nil
}

// GetTemplateByID retrieves a template by ID
func (r *InvoiceRepositoryNew) GetTemplateByID(id uint) (*models.InvoiceTemplate, error) {
	var template models.InvoiceTemplate

	if err := r.db.DB.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get template: %v", err)
	}

	return &template, nil
}

// CreateTemplate creates a new invoice template
func (r *InvoiceRepositoryNew) CreateTemplate(template *models.InvoiceTemplate) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		// If this is set as default, unset all other defaults
		if template.IsDefault {
			if err := tx.Model(&models.InvoiceTemplate{}).
				Where("is_default = ?", true).
				Update("is_default", false).Error; err != nil {
				return fmt.Errorf("failed to unset existing defaults: %v", err)
			}
		}

		// Create the template
		if err := tx.Create(template).Error; err != nil {
			return fmt.Errorf("failed to create template: %v", err)
		}

		return nil
	})
}

// UpdateTemplate updates an existing invoice template
func (r *InvoiceRepositoryNew) UpdateTemplate(template *models.InvoiceTemplate) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		// If this is set as default, unset all other defaults
		if template.IsDefault {
			if err := tx.Model(&models.InvoiceTemplate{}).
				Where("template_id != ? AND is_default = ?", template.TemplateID, true).
				Update("is_default", false).Error; err != nil {
				return fmt.Errorf("failed to unset existing defaults: %v", err)
			}
		}

		// Update the template
		result := tx.Model(&models.InvoiceTemplate{}).
			Where("template_id = ?", template.TemplateID).
			Updates(template)

		if result.Error != nil {
			return fmt.Errorf("failed to update template: %v", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("template with ID %d not found", template.TemplateID)
		}

		return nil
	})
}

// DeleteTemplate deletes an invoice template
func (r *InvoiceRepositoryNew) DeleteTemplate(id uint) error {
	// Check if template is in use
	var count int64
	if err := r.db.DB.Model(&models.Invoice{}).
		Where("template_id = ?", id).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check template usage: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete template that is in use by %d invoices", count)
	}

	result := r.db.DB.Delete(&models.InvoiceTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete template: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("template with ID %d not found", id)
	}

	return nil
}

// SetDefaultTemplate sets a template as the default
func (r *InvoiceRepositoryNew) SetDefaultTemplate(id uint) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		// Unset all existing defaults
		if err := tx.Model(&models.InvoiceTemplate{}).
			Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset existing defaults: %v", err)
		}

		// Set the new default
		result := tx.Model(&models.InvoiceTemplate{}).
			Where("template_id = ?", id).
			Update("is_default", true)

		if result.Error != nil {
			return fmt.Errorf("failed to set default template: %v", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("template with ID %d not found", id)
		}

		return nil
	})
}

// GetDefaultTemplate returns the default template
func (r *InvoiceRepositoryNew) GetDefaultTemplate() (*models.InvoiceTemplate, error) {
	var template models.InvoiceTemplate

	if err := r.db.DB.Where("is_default = ?", true).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default template found")
		}
		return nil, fmt.Errorf("failed to get default template: %v", err)
	}

	return &template, nil
}

// ================================================================
// COMPANY SETTINGS
// ================================================================

// GetCompanySettings returns the company settings
func (r *InvoiceRepositoryNew) GetCompanySettings() (*models.CompanySettings, error) {
	var settings models.CompanySettings

	if err := r.db.DB.First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings
			defaultSettings := &models.CompanySettings{
				CompanyName: "RentalCore Company",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := r.db.DB.Create(defaultSettings).Error; err != nil {
				return nil, fmt.Errorf("failed to create default company settings: %v", err)
			}
			return defaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get company settings: %v", err)
	}

	return &settings, nil
}

// UpdateCompanySettings updates the company settings
func (r *InvoiceRepositoryNew) UpdateCompanySettings(settings *models.CompanySettings) error {
	settings.UpdatedAt = time.Now()

	if err := r.db.DB.Save(settings).Error; err != nil {
		return fmt.Errorf("failed to update company settings: %v", err)
	}

	return nil
}

// ================================================================
// INVOICE SETTINGS
// ================================================================

// GetInvoiceSetting returns a specific invoice setting
func (r *InvoiceRepositoryNew) GetInvoiceSetting(key string) (string, error) {
	var setting models.InvoiceSetting

	if err := r.db.DB.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil // Return empty string for missing settings
		}
		return "", fmt.Errorf("failed to get invoice setting: %v", err)
	}

	if setting.SettingValue != nil {
		return *setting.SettingValue, nil
	}

	return "", nil
}

// getSettingWithDefault returns a setting value or a default if not found
func (r *InvoiceRepositoryNew) getSettingWithDefault(key, defaultValue string) string {
	value, err := r.GetInvoiceSetting(key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

// GetAllInvoiceSettings returns all invoice settings as a structured object
func (r *InvoiceRepositoryNew) GetAllInvoiceSettings() (*models.InvoiceSettings, error) {
	var dbSettings []models.InvoiceSetting

	if err := r.db.DB.Find(&dbSettings).Error; err != nil {
		return nil, fmt.Errorf("failed to get invoice settings: %v", err)
	}

	// Create settings with defaults
	settings := &models.InvoiceSettings{
		InvoiceNumberPrefix:     "RE",
		InvoiceNumberFormat:     "{prefix}{sequence:4}",
		DefaultPaymentTerms:     30,
		DefaultTaxRate:          19.0,
		AutoCalculateRentalDays: true,
		ShowLogoOnInvoice:       true,
		CurrencySymbol:          "€",
		CurrencyCode:            "EUR",
		DateFormat:              "DD.MM.YYYY",
	}

	// Override with database values
	for _, setting := range dbSettings {
		if setting.SettingValue == nil {
			continue
		}

		switch setting.SettingKey {
		case "invoice_number_prefix":
			settings.InvoiceNumberPrefix = *setting.SettingValue
		case "invoice_number_format":
			settings.InvoiceNumberFormat = *setting.SettingValue
		case "default_payment_terms":
			if val, err := strconv.Atoi(*setting.SettingValue); err == nil {
				settings.DefaultPaymentTerms = val
			}
		case "default_tax_rate":
			if val, err := strconv.ParseFloat(*setting.SettingValue, 64); err == nil {
				settings.DefaultTaxRate = val
			}
		case "auto_calculate_rental_days":
			settings.AutoCalculateRentalDays = *setting.SettingValue == "true"
		case "show_logo_on_invoice":
			settings.ShowLogoOnInvoice = *setting.SettingValue == "true"
		case "currency_symbol":
			settings.CurrencySymbol = *setting.SettingValue
		case "currency_code":
			settings.CurrencyCode = *setting.SettingValue
		case "date_format":
			settings.DateFormat = *setting.SettingValue
		}
	}

	return settings, nil
}

// UpdateInvoiceSetting updates a specific invoice setting
func (r *InvoiceRepositoryNew) UpdateInvoiceSetting(key, value string, updatedBy *uint) error {
	setting := models.InvoiceSetting{
		SettingKey:   key,
		SettingValue: &value,
		UpdatedBy:    updatedBy,
		UpdatedAt:    time.Now(),
	}

	if err := r.db.DB.Save(&setting).Error; err != nil {
		return fmt.Errorf("failed to update invoice setting: %v", err)
	}

	return nil
}

// ================================================================
// STATISTICS
// ================================================================

// GetInvoiceStats returns invoice statistics
func (r *InvoiceRepositoryNew) GetInvoiceStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total invoices
	var totalInvoices int64
	r.db.DB.Model(&models.Invoice{}).Where("status != 'cancelled'").Count(&totalInvoices)
	stats["total_invoices"] = totalInvoices

	// Total revenue (paid invoices)
	var totalRevenue float64
	r.db.DB.Model(&models.Invoice{}).
		Where("status = 'paid'").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue)
	stats["total_revenue"] = totalRevenue

	// Outstanding amount
	var outstanding float64
	r.db.DB.Model(&models.Invoice{}).
		Where("status NOT IN ('paid', 'cancelled')").
		Select("COALESCE(SUM(balance_due), 0)").
		Scan(&outstanding)
	stats["outstanding_amount"] = outstanding

	// Overdue invoices
	var overdueCount int64
	r.db.DB.Model(&models.Invoice{}).
		Where("due_date < ? AND status NOT IN ('paid', 'cancelled')", time.Now()).
		Count(&overdueCount)
	stats["overdue_count"] = overdueCount

	// Draft invoices
	var draftCount int64
	r.db.DB.Model(&models.Invoice{}).
		Where("status = 'draft'").
		Count(&draftCount)
	stats["draft_count"] = draftCount

	return stats, nil
}