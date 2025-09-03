package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorHandler provides centralized error handling and recovery
type ErrorHandler struct{}

func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// SafeHTML safely renders HTML templates with proper error handling
// This prevents blank pages by ensuring proper context and fallback handling
func SafeHTML(c *gin.Context, statusCode int, templateName string, data gin.H) {
	// Ensure user context is always available for base template
	if data == nil {
		data = gin.H{}
	}
	
	// Get user context if not already provided
	if _, exists := data["user"]; !exists {
		user, _ := GetCurrentUser(c)
		data["user"] = user
	}
	
	// Ensure title is always provided
	if _, exists := data["title"]; !exists {
		data["title"] = "TS Jobscanner"
	}
	
	// Attempt to render the template
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SafeHTML: Template rendering panic for %s: %v", templateName, r)
			renderErrorPage(c, http.StatusInternalServerError, "Template rendering error", data["user"])
		}
	}()
	
	log.Printf("SafeHTML: Rendering template %s with status %d", templateName, statusCode)
	c.HTML(statusCode, templateName, data)
}

// SafeRedirect safely redirects with proper logging
func SafeRedirect(c *gin.Context, statusCode int, location string) {
	log.Printf("SafeRedirect: Redirecting to %s with status %d", location, statusCode)
	c.Redirect(statusCode, location)
}

// SafeJSON safely renders JSON with proper error handling
func SafeJSON(c *gin.Context, statusCode int, data interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SafeJSON: JSON rendering panic: %v", r)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
				"code":  "RENDER_ERROR",
			})
		}
	}()
	
	log.Printf("SafeJSON: Rendering JSON with status %d", statusCode)
	c.JSON(statusCode, data)
}

// renderErrorPage renders a safe error page that should never fail
func renderErrorPage(c *gin.Context, statusCode int, message string, user interface{}) {
	log.Printf("renderErrorPage: Rendering error page - Status: %d, Message: %s", statusCode, message)
	
	// Check if response has already been written
	if c.Writer.Written() {
		log.Printf("renderErrorPage: Response already written, skipping error page")
		return
	}
	
	// Get request ID for debugging
	requestID := ""
	if id, exists := c.Get("request_id"); exists {
		requestID = id.(string)
	}
	
	// Try to use the enhanced error template first
	defer func() {
		if r := recover(); r != nil {
			log.Printf("renderErrorPage: Error template also failed: %v", r)
			// Check if response has already been written after panic
			if c.Writer.Written() {
				log.Printf("renderErrorPage: Response already written after panic, cannot render fallback")
				return
			}
			// Last resort: plain HTML response
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(statusCode, `
<!DOCTYPE html>
<html>
<head>
    <title>Error %d - RentalCore</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
    <div class="container mt-5">
        <div class="alert alert-danger">
            <h4>Application Error %d</h4>
            <p>%s</p>
            <p><a href="/" class="btn btn-primary">Return Home</a></p>
            <small class="text-muted">Request ID: %s</small>
        </div>
    </div>
</body>
</html>`, statusCode, statusCode, message, requestID)
		}
	}()
	
	// Try to render enhanced error template first
	errorData := gin.H{
		"error_code":    statusCode,
		"error_message": getErrorMessage(statusCode, message),
		"error_details": message,
		"request_id":    requestID,
		"timestamp":     time.Now().Format("2006-01-02 15:04:05"),
		"user":          user,
	}
	
	// Ensure user is not nil to prevent template errors
	if user == nil {
		errorData["user"] = gin.H{"Username": "Guest", "FirstName": "", "LastName": ""}
	}
	
	c.HTML(statusCode, "error_page.html", errorData)
}

// getErrorMessage returns a user-friendly error message based on status code
func getErrorMessage(statusCode int, originalMessage string) string {
	switch statusCode {
	case 400:
		return "Bad Request - The request was invalid or cannot be processed"
	case 401:
		return "Unauthorized - You need to log in to access this page"
	case 403:
		return "Forbidden - You don't have permission to access this resource"
	case 404:
		return "Page Not Found - The requested page could not be found"
	case 500:
		return "Internal Server Error - Something went wrong on the server"
	case 502:
		return "Bad Gateway - The server received an invalid response"
	case 503:
		return "Service Unavailable - The server is temporarily unavailable"
	default:
		if originalMessage != "" {
			return originalMessage
		}
		return "An unexpected error occurred"
	}
}

// GlobalErrorHandler provides global error recovery middleware
func GlobalErrorHandler() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(gin.DefaultWriter, func(c *gin.Context, recovered interface{}) {
		log.Printf("GlobalErrorHandler: Panic recovered: %v", recovered)
		
		// Get user context for error page
		user, _ := GetCurrentUser(c)
		
		// Render safe error page
		renderErrorPage(c, http.StatusInternalServerError, "An unexpected error occurred", user)
	})
}

// NotFoundHandler handles 404 errors with proper template rendering
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("NotFoundHandler: 404 for path: %s", c.Request.URL.Path)
		
		user, _ := GetCurrentUser(c)
		
		// Check if this is an API request
		if c.GetHeader("Accept") == "application/json" || 
		   c.ContentType() == "application/json" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Resource not found",
				"path":  c.Request.URL.Path,
			})
			return
		}
		
		// Render 404 page
		renderErrorPage(c, http.StatusNotFound, "Page not found", user)
	}
}

// TemplateExistsCheck verifies template exists before rendering
func TemplateExistsCheck(templateName string) bool {
	// This is a simple check - in production you might want to maintain
	// a registry of available templates
	commonTemplates := map[string]bool{
		"base.html":                       true,
		"error.html":                      true,
		"error_standalone.html":           true,
		"error_page.html":                 true,
		"login.html":                      true,
		"home.html":                   true,
		"cases_list.html":                 true,
		"case_form.html":                  true,
		"case_detail.html":                true,
		"case_management_simple.html":     true,
		"customers.html":              true,
		"customer_form.html":          true,
		"customer_detail.html":            true,
		"devices.html":                true,
		"device_form_new.html":            true,
		"device_detail.html":              true,
		"jobs_new.html":                   true,
		"job_form.html":               true,
		"job_detail.html":                 true,
		"equipment_packages_standalone.html": true,
		"equipment_package_form.html":     true,
		"equipment_package_detail.html":   true,
		"bulk_operations.html":            true,
		"documents_list.html":             true,
		"document_upload_form.html":       true,
		"signature_form.html":             true,
		"financial_dashboard.html":        true,
		"transactions_list.html":          true,
		"transaction_form.html":           true,
		"transaction_detail.html":         true,
		"financial_reports.html":          true,
		"users_list.html":                 true,
		"user_form.html":                  true,
		"user_detail.html":                true,
		"analytics_dashboard.html":        true,
		"search_results.html":             true,
		"scan_select_job.html":            true,
		"scan_job.html":                   true,
		"mobile_scanner.html":             true,
		"mobile_scanner_enhanced.html":    true,
		"security_roles.html":             true,
		"security_audit.html":             true,
		"profile_settings.html":           true,
		"devices_standalone.html":         true,
		"invoices_list.html":              true,
		"invoices_content.html":           true,
		"company_settings.html":           true,
		"monitoring_dashboard.html":       true,
	}
	
	return commonTemplates[templateName]
}

// LogTemplateRender logs template rendering for debugging
func LogTemplateRender(templateName string, data gin.H) {
	log.Printf("Template Render: %s with data keys: %v", templateName, getKeys(data))
}

// getKeys returns the keys of a gin.H map for logging
func getKeys(data gin.H) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}