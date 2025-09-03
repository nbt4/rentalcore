package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CompanyHandler struct {
	db *gorm.DB
}

func NewCompanyHandler(db *gorm.DB) *CompanyHandler {
	return &CompanyHandler{
		db: db,
	}
}

// CompanySettingsForm displays the company settings form
func (h *CompanyHandler) CompanySettingsForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get current company settings
	company, err := h.getCompanySettings()
	if err != nil {
		log.Printf("CompanySettingsForm: Error fetching company settings: %v", err)
		// Create default empty company settings
		company = &models.CompanySettings{
			CompanyName: "Ihre Firma GmbH",
		}
	}

	log.Printf("DEBUG: CompanySettingsForm handler called successfully - rendering company_settings.html")
	
	// Check for success message
	var successMsg string
	if c.Query("success") == "1" {
		successMsg = "Company settings saved successfully!"
	}
	
	c.HTML(http.StatusOK, "company_settings.html", gin.H{
		"title":        "Company Settings",
		"user":         user,
		"company":      company,
		"success":      successMsg,
		"currentPage":  "settings",
	})
}

// UpdateCompanySettingsForm handles form-based company settings updates
func (h *CompanyHandler) UpdateCompanySettingsForm(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get form values
	companyName := c.PostForm("company_name")
	taxNumber := c.PostForm("tax_number")
	email := c.PostForm("email")
	phone := c.PostForm("phone")
	addressLine1 := c.PostForm("address_line1")
	addressLine2 := c.PostForm("address_line2")
	city := c.PostForm("city")
	postalCode := c.PostForm("postal_code")
	country := c.PostForm("country")

	// Validate required fields
	if strings.TrimSpace(companyName) == "" {
		log.Printf("UpdateCompanySettingsForm: Company name is required")
		c.HTML(http.StatusBadRequest, "company_settings.html", gin.H{
			"title":   "Company Settings",
			"user":    user,
			"company": nil,
			"error":   "Company name is required",
		})
		return
	}

	// Get existing company settings or create new
	company, err := h.getCompanySettings()
	if err != nil {
		// Create new company settings
		company = &models.CompanySettings{}
	}

	// Update fields
	company.CompanyName = strings.TrimSpace(companyName)
	company.TaxNumber = h.trimStringPointer(&taxNumber)
	company.Email = h.trimStringPointer(&email)
	company.Phone = h.trimStringPointer(&phone)
	company.AddressLine1 = h.trimStringPointer(&addressLine1)
	company.AddressLine2 = h.trimStringPointer(&addressLine2)
	company.City = h.trimStringPointer(&city)
	company.PostalCode = h.trimStringPointer(&postalCode)
	company.Country = h.trimStringPointer(&country)
	company.UpdatedAt = time.Now()

	// Save to database
	var result *gorm.DB
	if company.ID == 0 {
		company.CreatedAt = time.Now()
		result = h.db.Create(company)
	} else {
		result = h.db.Save(company)
	}

	if result.Error != nil {
		log.Printf("UpdateCompanySettingsForm: Database error: %v", result.Error)
		c.HTML(http.StatusInternalServerError, "company_settings.html", gin.H{
			"title":   "Company Settings",
			"user":    user,
			"company": company,
			"error":   "Failed to save company settings: " + result.Error.Error(),
		})
		return
	}

	log.Printf("Company settings updated successfully by user %s", user.Username)
	c.Redirect(http.StatusSeeOther, "/settings/company?success=1")
}

// GetCompanySettings returns company settings as JSON
func (h *CompanyHandler) GetCompanySettings(c *gin.Context) {
	company, err := h.getCompanySettings()
	if err != nil {
		log.Printf("GetCompanySettings: Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load company settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"company": company,
	})
}

// UpdateCompanySettings updates company settings
func (h *CompanyHandler) UpdateCompanySettings(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var request models.CompanySettings
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("UpdateCompanySettings: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if strings.TrimSpace(request.CompanyName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Company name is required",
		})
		return
	}

	// Get existing company settings or create new
	company, err := h.getCompanySettings()
	if err != nil {
		// Create new company settings
		company = &models.CompanySettings{}
	}

	// Update basic fields
	company.CompanyName = strings.TrimSpace(request.CompanyName)
	company.AddressLine1 = h.trimStringPointer(request.AddressLine1)
	company.AddressLine2 = h.trimStringPointer(request.AddressLine2)
	company.City = h.trimStringPointer(request.City)
	company.State = h.trimStringPointer(request.State)
	company.PostalCode = h.trimStringPointer(request.PostalCode)
	company.Country = h.trimStringPointer(request.Country)
	company.Phone = h.trimStringPointer(request.Phone)
	company.Email = h.trimStringPointer(request.Email)
	company.Website = h.trimStringPointer(request.Website)
	company.TaxNumber = h.trimStringPointer(request.TaxNumber)
	company.VATNumber = h.trimStringPointer(request.VATNumber)
	
	// Update German banking fields
	company.BankName = h.trimStringPointer(request.BankName)
	company.IBAN = h.trimStringPointer(request.IBAN)
	company.BIC = h.trimStringPointer(request.BIC)
	company.AccountHolder = h.trimStringPointer(request.AccountHolder)
	
	// Update German legal fields
	company.CEOName = h.trimStringPointer(request.CEOName)
	company.RegisterCourt = h.trimStringPointer(request.RegisterCourt)
	company.RegisterNumber = h.trimStringPointer(request.RegisterNumber)
	
	// Update invoice text fields
	company.FooterText = h.trimStringPointer(request.FooterText)
	company.PaymentTermsText = h.trimStringPointer(request.PaymentTermsText)
	
	// Update email settings
	company.SMTPHost = h.trimStringPointer(request.SMTPHost)
	company.SMTPPort = request.SMTPPort
	company.SMTPUsername = h.trimStringPointer(request.SMTPUsername)
	company.SMTPPassword = h.trimStringPointer(request.SMTPPassword)
	company.SMTPFromEmail = h.trimStringPointer(request.SMTPFromEmail)
	company.SMTPFromName = h.trimStringPointer(request.SMTPFromName)
	company.SMTPUseTLS = request.SMTPUseTLS
	
	company.UpdatedAt = time.Now()

	// If updating existing record, preserve logo path if not provided
	if request.LogoPath != nil && *request.LogoPath != "" {
		company.LogoPath = request.LogoPath
	}

	// Save to database
	var result *gorm.DB
	if company.ID == 0 {
		company.CreatedAt = time.Now()
		result = h.db.Create(company)
	} else {
		result = h.db.Save(company)
	}

	if result.Error != nil {
		log.Printf("UpdateCompanySettings: Database error: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save company settings",
			"details": result.Error.Error(),
		})
		return
	}

	log.Printf("Company settings updated successfully by user %s", user.Username)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Company settings updated successfully",
		"company": company,
	})
}

// UploadCompanyLogo handles company logo upload
func (h *CompanyHandler) UploadCompanyLogo(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Parse multipart form with 2MB max memory
	if err := c.Request.ParseMultipartForm(2 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No logo file provided"})
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be an image"})
		return
	}

	// Validate file size (max 2MB)
	if header.Size > 2<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size must be less than 2MB"})
		return
	}

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads/logos"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("UploadCompanyLogo: Failed to create uploads directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png" // Default extension
	}
	filename := fmt.Sprintf("company_logo_%d_%s%s", timestamp, user.Username, ext)
	filePath := filepath.Join(uploadsDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("UploadCompanyLogo: Failed to create destination file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("UploadCompanyLogo: Failed to copy file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Generate web-accessible path
	webPath := "/" + strings.ReplaceAll(filePath, "\\", "/")

	// Update company settings with new logo path
	company, err := h.getCompanySettings()
	if err != nil {
		// Create new company settings if none exist
		company = &models.CompanySettings{
			CompanyName: "Ihre Firma GmbH",
			CreatedAt:   time.Now(),
		}
	}

	// Remove old logo file if exists
	if company.LogoPath != nil && *company.LogoPath != "" {
		oldPath := strings.TrimPrefix(*company.LogoPath, "/")
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Remove(oldPath); err != nil {
				log.Printf("UploadCompanyLogo: Failed to remove old logo: %v", err)
			}
		}
	}

	company.LogoPath = &webPath
	company.UpdatedAt = time.Now()

	// Save to database
	var result *gorm.DB
	if company.ID == 0 {
		result = h.db.Create(company)
	} else {
		result = h.db.Save(company)
	}

	if result.Error != nil {
		log.Printf("UploadCompanyLogo: Database error: %v", result.Error)
		// Clean up uploaded file on database error
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save logo path",
			"details": result.Error.Error(),
		})
		return
	}

	log.Printf("Company logo uploaded successfully by user %s: %s", user.Username, filename)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Logo uploaded successfully",
		"logoPath": webPath,
	})
}

// DeleteCompanyLogo removes the company logo
func (h *CompanyHandler) DeleteCompanyLogo(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get current company settings
	company, err := h.getCompanySettings()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company settings not found"})
		return
	}

	// Check if logo exists
	if company.LogoPath == nil || *company.LogoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No logo to delete"})
		return
	}

	// Remove logo file
	oldPath := strings.TrimPrefix(*company.LogoPath, "/")
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			log.Printf("DeleteCompanyLogo: Failed to remove logo file: %v", err)
		}
	}

	// Update database
	company.LogoPath = nil
	company.UpdatedAt = time.Now()

	if err := h.db.Save(company).Error; err != nil {
		log.Printf("DeleteCompanyLogo: Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update company settings",
			"details": err.Error(),
		})
		return
	}

	log.Printf("Company logo deleted successfully by user %s", user.Username)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logo deleted successfully",
	})
}

// Helper methods

func (h *CompanyHandler) getCompanySettings() (*models.CompanySettings, error) {
	var company models.CompanySettings
	
	// Try to get the first (and should be only) company settings record
	result := h.db.First(&company)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("company settings not found")
		}
		return nil, result.Error
	}
	
	return &company, nil
}

func (h *CompanyHandler) trimStringPointer(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// CompanySettingsAPI provides API access to company settings
func (h *CompanyHandler) CompanySettingsAPI(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		h.GetCompanySettings(c)
	case "PUT":
		h.UpdateCompanySettings(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

// SMTP Configuration handlers
func (h *CompanyHandler) GetSMTPConfig(c *gin.Context) {
	_, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get company settings with email configuration
	company, err := h.getCompanySettings()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"config": gin.H{
				"smtp_host":       "",
				"smtp_port":       587,
				"smtp_username":   "",
				"smtp_from_email": "",
				"smtp_from_name":  "",
				"smtp_use_tls":    true,
			},
		})
		return
	}

	// For security, we don't return the actual password
	config := gin.H{
		"smtp_host":       h.getStringValue(company.SMTPHost),
		"smtp_port":       h.getIntValue(company.SMTPPort, 587),
		"smtp_username":   h.getStringValue(company.SMTPUsername),
		"smtp_from_email": h.getStringValue(company.SMTPFromEmail),
		"smtp_from_name":  h.getStringValue(company.SMTPFromName),
		"smtp_use_tls":    h.getBoolValue(company.SMTPUseTLS, true),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  config,
	})
}

func (h *CompanyHandler) UpdateSMTPConfig(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var request struct {
		SMTPHost      string `json:"smtp_host" binding:"required"`
		SMTPPort      int    `json:"smtp_port" binding:"required"`
		SMTPUsername  string `json:"smtp_username" binding:"required"`
		SMTPPassword  string `json:"smtp_password"`
		SMTPFromEmail string `json:"smtp_from_email" binding:"required"`
		SMTPFromName  string `json:"smtp_from_name"`
		SMTPUseTLS    bool   `json:"smtp_use_tls"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input data",
			"details": err.Error(),
		})
		return
	}

	// Get existing company settings or create new
	company, err := h.getCompanySettings()
	if err != nil {
		company = &models.CompanySettings{
			CompanyName: "Ihre Firma GmbH",
			CreatedAt:   time.Now(),
		}
	}

	// Update email settings
	company.SMTPHost = &request.SMTPHost
	company.SMTPPort = &request.SMTPPort
	company.SMTPUsername = &request.SMTPUsername
	company.SMTPFromEmail = &request.SMTPFromEmail
	company.SMTPUseTLS = &request.SMTPUseTLS
	
	if request.SMTPFromName != "" {
		company.SMTPFromName = &request.SMTPFromName
	}
	
	// Only update password if provided
	if request.SMTPPassword != "" {
		company.SMTPPassword = &request.SMTPPassword
	}
	
	company.UpdatedAt = time.Now()

	// Save to database
	var result *gorm.DB
	if company.ID == 0 {
		result = h.db.Create(company)
	} else {
		result = h.db.Save(company)
	}

	if result.Error != nil {
		log.Printf("UpdateSMTPConfig: Database error: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save email configuration",
			"details": result.Error.Error(),
		})
		return
	}

	log.Printf("SMTP config updated successfully by user %s: %s:%d", user.Username, request.SMTPHost, request.SMTPPort)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email configuration updated successfully",
	})
}

func (h *CompanyHandler) TestSMTPConnection(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get company settings with email configuration
	company, err := h.getCompanySettings()
	if err != nil || company.SMTPHost == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email configuration not found. Please configure email settings first.",
		})
		return
	}

	// Test email configuration by sending a test email
	testEmail := c.Query("test_email")
	if testEmail == "" {
		testEmail = user.Email
	}

	// Create email configuration from company settings
	emailConfig := &config.EmailConfig{
		SMTPHost:     h.getStringValue(company.SMTPHost),
		SMTPPort:     h.getIntValue(company.SMTPPort, 587),
		SMTPUsername: h.getStringValue(company.SMTPUsername),
		SMTPPassword: h.getStringValue(company.SMTPPassword),
		FromEmail:    h.getStringValue(company.SMTPFromEmail),
		FromName:     h.getStringValue(company.SMTPFromName),
		UseTLS:       h.getBoolValue(company.SMTPUseTLS, true),
	}

	// Create email service and send test email
	emailService := services.NewEmailService(emailConfig)
	testData := &services.EmailData{
		Company: company,
	}

	err = emailService.SendTestEmail(testEmail, testData)
	if err != nil {
		log.Printf("TestSMTPConnection: Failed to send test email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send test email",
			"details": err.Error(),
		})
		return
	}

	log.Printf("SMTP connection test successful by user %s, test email sent to %s", user.Username, testEmail)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Test email sent successfully to %s", testEmail),
	})
}

// Helper methods for safe pointer access
func (h *CompanyHandler) getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func (h *CompanyHandler) getIntValue(ptr *int, defaultValue int) int {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

func (h *CompanyHandler) getBoolValue(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}