package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SecurityHandler struct {
	db *gorm.DB
}

func NewSecurityHandler(db *gorm.DB) *SecurityHandler {
	return &SecurityHandler{db: db}
}

// Permission represents a permission with friendly name and description
type Permission struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// GetPermissionDefinitions returns all available permissions with descriptions
func (h *SecurityHandler) GetPermissionDefinitions() []Permission {
	return []Permission{
		// User Management
		{Code: "users.manage", Name: "Manage Users", Description: "Create, edit, and delete user accounts", Category: "User Management"},
		{Code: "users.view", Name: "View Users", Description: "View user lists and basic information", Category: "User Management"},
		{Code: "users.create", Name: "Create Users", Description: "Create new user accounts", Category: "User Management"},
		{Code: "users.edit", Name: "Edit Users", Description: "Modify existing user information", Category: "User Management"},
		{Code: "users.delete", Name: "Delete Users", Description: "Remove user accounts", Category: "User Management"},
		
		// Job Management
		{Code: "jobs.manage", Name: "Manage Jobs", Description: "Full access to job creation and management", Category: "Job Management"},
		{Code: "jobs.view", Name: "View Jobs", Description: "View job listings and details", Category: "Job Management"},
		{Code: "jobs.create", Name: "Create Jobs", Description: "Create new jobs and projects", Category: "Job Management"},
		{Code: "jobs.edit", Name: "Edit Jobs", Description: "Modify existing job information", Category: "Job Management"},
		{Code: "jobs.delete", Name: "Delete Jobs", Description: "Remove jobs from the system", Category: "Job Management"},
		
		// Device Management
		{Code: "devices.manage", Name: "Manage Equipment", Description: "Add, edit, and track equipment inventory", Category: "Equipment"},
		{Code: "devices.view", Name: "View Equipment", Description: "View equipment lists and availability", Category: "Equipment"},
		{Code: "devices.create", Name: "Add Equipment", Description: "Add new equipment to inventory", Category: "Equipment"},
		{Code: "devices.edit", Name: "Edit Equipment", Description: "Modify equipment information", Category: "Equipment"},
		{Code: "devices.delete", Name: "Remove Equipment", Description: "Remove equipment from inventory", Category: "Equipment"},
		
		// Customer Management
		{Code: "customers.manage", Name: "Manage Customers", Description: "Create and edit customer information", Category: "Customer Management"},
		{Code: "customers.view", Name: "View Customers", Description: "View customer listings and details", Category: "Customer Management"},
		{Code: "customers.create", Name: "Create Customers", Description: "Add new customers to database", Category: "Customer Management"},
		{Code: "customers.edit", Name: "Edit Customers", Description: "Modify customer information", Category: "Customer Management"},
		{Code: "customers.delete", Name: "Delete Customers", Description: "Remove customers from database", Category: "Customer Management"},
		
		// Reports & Analytics
		{Code: "reports.view", Name: "View Reports", Description: "Access analytics and generate reports", Category: "Reports & Analytics"},
		{Code: "analytics.view", Name: "View Analytics", Description: "Access dashboard analytics and insights", Category: "Reports & Analytics"},
		{Code: "analytics.export", Name: "Export Data", Description: "Export analytics data and reports", Category: "Reports & Analytics"},
		
		// System Settings
		{Code: "settings.manage", Name: "System Settings", Description: "Configure application settings", Category: "System"},
		{Code: "roles.manage", Name: "Manage Roles", Description: "Create and modify user roles and permissions", Category: "System"},
		{Code: "audit.view", Name: "View Audit Logs", Description: "Access system audit trail and logs", Category: "System"},
		
		// Scanner & Mobile
		{Code: "scan.use", Name: "Use Scanner", Description: "Access mobile barcode scanning features", Category: "Scanner & Mobile"},
		{Code: "mobile.access", Name: "Mobile Access", Description: "Access mobile app features", Category: "Scanner & Mobile"},
		
		// Documents
		{Code: "documents.manage", Name: "Manage Documents", Description: "Upload, view, and organize documents", Category: "Documents"},
		{Code: "documents.view", Name: "View Documents", Description: "View and download documents", Category: "Documents"},
		{Code: "documents.upload", Name: "Upload Documents", Description: "Upload new documents and files", Category: "Documents"},
		{Code: "documents.sign", Name: "Digital Signatures", Description: "Create and verify digital signatures", Category: "Documents"},
		
		// Financial
		{Code: "financial.view", Name: "View Financial Data", Description: "Access financial reports and transactions", Category: "Financial"},
		{Code: "financial.manage", Name: "Manage Finances", Description: "Create invoices and manage transactions", Category: "Financial"},
		{Code: "invoices.generate", Name: "Generate Invoices", Description: "Create and send customer invoices", Category: "Financial"},
		
		// Super Admin
		{Code: "*", Name: "Full System Access", Description: "Complete administrative access to all features", Category: "Super Admin"},
	}
}

// GetPermissionDefinitionsAPI returns permission definitions for API calls
func (h *SecurityHandler) GetPermissionDefinitionsAPI(c *gin.Context) {
	if !h.hasPermission(c, "roles.manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	permissions := h.GetPermissionDefinitions()
	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

// ================================================================
// ROLE MANAGEMENT
// ================================================================

// GetRoles returns all available roles
func (h *SecurityHandler) GetRoles(c *gin.Context) {
	if !h.hasPermission(c, "role.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var roles []models.Role
	result := h.db.Preload("UserRoles").Where("is_active = ?", true).Find(&roles)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// GetRole returns a specific role by ID
func (h *SecurityHandler) GetRole(c *gin.Context) {
	if !h.hasPermission(c, "role.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	roleID := c.Param("id")
	var role models.Role
	result := h.db.Preload("UserRoles").First(&role, roleID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"role": role})
}

// CreateRole creates a new role
func (h *SecurityHandler) CreateRole(c *gin.Context) {
	if !h.hasPermission(c, "role.create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	role.IsActive = true
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	result := h.db.Create(&role)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	// Log the action
	h.logAction(c, "create", "role", fmt.Sprintf("%d", role.RoleID), nil, role)

	c.JSON(http.StatusCreated, gin.H{"role": role})
}

// UpdateRole updates an existing role
func (h *SecurityHandler) UpdateRole(c *gin.Context) {
	if !h.hasPermission(c, "role.update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	roleID := c.Param("id")
	var role models.Role
	result := h.db.First(&role, roleID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	// Check if it's a system role
	if role.IsSystemRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system roles"})
		return
	}

	oldRole := role // Store for audit log

	var updateData models.Role
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update allowed fields
	role.DisplayName = updateData.DisplayName
	role.Description = updateData.Description
	role.Permissions = updateData.Permissions
	role.IsActive = updateData.IsActive
	role.UpdatedAt = time.Now()

	result = h.db.Save(&role)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	// Log the action
	h.logAction(c, "update", "role", fmt.Sprintf("%d", role.RoleID), oldRole, role)

	c.JSON(http.StatusOK, gin.H{"role": role})
}

// DeleteRole deactivates a role
func (h *SecurityHandler) DeleteRole(c *gin.Context) {
	if !h.hasPermission(c, "role.delete") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	roleID := c.Param("id")
	var role models.Role
	result := h.db.First(&role, roleID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	// Check if it's a system role
	if role.IsSystemRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system roles"})
		return
	}

	oldRole := role // Store for audit log

	// Deactivate role instead of deleting
	role.IsActive = false
	role.UpdatedAt = time.Now()

	result = h.db.Save(&role)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate role"})
		return
	}

	// Log the action
	h.logAction(c, "delete", "role", fmt.Sprintf("%d", role.RoleID), oldRole, role)

	c.JSON(http.StatusOK, gin.H{"message": "Role deactivated successfully"})
}

// ================================================================
// USER ROLE MANAGEMENT
// ================================================================

// GetUserRoles returns roles assigned to a user
func (h *SecurityHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("userId")
	
	// Users can view their own roles, admins can view any user's roles
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if fmt.Sprintf("%d", currentUser.UserID) != userID && !h.hasPermission(c, "user.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var userRoles []models.UserRole
	result := h.db.Preload("Role").Where("userID = ? AND is_active = ?", userID, true).Find(&userRoles)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"userRoles": userRoles})
}

// AssignUserRole assigns a role to a user
func (h *SecurityHandler) AssignUserRole(c *gin.Context) {
	if !h.hasPermission(c, "user.assign_role") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	userID := c.Param("userId")
	
	var request struct {
		RoleID    uint       `json:"roleId" binding:"required"`
		ExpiresAt *time.Time `json:"expiresAt"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Check if user exists
	var user models.User
	result := h.db.First(&user, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Check if role exists and is active
	var role models.Role
	result = h.db.Where("roleID = ? AND is_active = ?", request.RoleID, true).First(&role)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found or inactive"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	// Check if user already has this role (active or inactive)
	var existing models.UserRole
	result = h.db.Where("userID = ? AND roleID = ?", userID, request.RoleID).First(&existing)
	if result.Error == nil {
		if existing.IsActive {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has this role"})
			return
		} else {
			// Reactivate existing role assignment
			existing.IsActive = true
			existing.AssignedAt = time.Now()
			existing.AssignedBy = &currentUser.UserID
			existing.ExpiresAt = request.ExpiresAt
			
			result = h.db.Save(&existing)
			if result.Error != nil {
				fmt.Printf("ERROR reactivating user role: %v\n", result.Error)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role", "details": result.Error.Error()})
				return
			}
			
			// Load role data for response
			h.db.Preload("Role").First(&existing, "userID = ? AND roleID = ?", userID, request.RoleID)
			
			// Log the action
			h.logAction(c, "assign_role", "user", userID, nil, existing)
			
			c.JSON(http.StatusCreated, gin.H{"userRole": existing})
			return
		}
	}

	// Create new user role assignment
	userRole := models.UserRole{
		UserID:     user.UserID,
		RoleID:     request.RoleID,
		AssignedAt: time.Now(),
		AssignedBy: &currentUser.UserID,
		ExpiresAt:  request.ExpiresAt,
		IsActive:   true,
	}

	result = h.db.Create(&userRole)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role"})
		return
	}

	// Load role data for response
	h.db.Preload("Role").First(&userRole, "userID = ? AND roleID = ?", userID, request.RoleID)

	// Log the action
	h.logAction(c, "assign_role", "user", userID, nil, userRole)

	c.JSON(http.StatusCreated, gin.H{"userRole": userRole})
}

// RevokeUserRole revokes a role from a user
func (h *SecurityHandler) RevokeUserRole(c *gin.Context) {
	if !h.hasPermission(c, "user.revoke_role") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	userID := c.Param("userId")
	roleID := c.Param("roleId")

	var userRole models.UserRole
	result := h.db.Where("userID = ? AND roleID = ? AND is_active = ?", userID, roleID, true).First(&userRole)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User role assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user role"})
		return
	}

	oldUserRole := userRole // Store for audit log

	// Deactivate the role assignment
	userRole.IsActive = false
	result = h.db.Save(&userRole)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke role"})
		return
	}

	// Log the action
	h.logAction(c, "revoke_role", "user", userID, oldUserRole, userRole)

	c.JSON(http.StatusOK, gin.H{"message": "Role revoked successfully"})
}

// ================================================================
// AUDIT LOG
// ================================================================

// GetAuditLogs returns audit logs with filtering and pagination
func (h *SecurityHandler) GetAuditLogs(c *gin.Context) {
	if !h.hasPermission(c, "audit.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// Parse query parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 50
	if ps := c.Query("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	// Build query
	query := h.db.Model(&models.AuditLog{}).Preload("User")

	// Apply filters
	if userID := c.Query("userId"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	if entityType := c.Query("entityType"); entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	if entityID := c.Query("entityId"); entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}

	if startDate := c.Query("startDate"); startDate != "" {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("timestamp >= ?", parsed)
		}
	}

	if endDate := c.Query("endDate"); endDate != "" {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("timestamp <= ?", parsed.Add(24*time.Hour))
		}
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var auditLogs []models.AuditLog
	result := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&auditLogs)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auditLogs": auditLogs,
		"pagination": gin.H{
			"page":      page,
			"pageSize":  pageSize,
			"total":     total,
			"totalPages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetAuditLog returns a specific audit log entry
func (h *SecurityHandler) GetAuditLog(c *gin.Context) {
	if !h.hasPermission(c, "audit.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	auditID := c.Param("id")
	var auditLog models.AuditLog
	result := h.db.Preload("User").First(&auditLog, auditID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Audit log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auditLog": auditLog})
}

// ExportAuditLogs exports audit logs to CSV format
func (h *SecurityHandler) ExportAuditLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	userID := c.Query("userId")
	action := c.Query("action")
	entityType := c.Query("entityType")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only CSV format is supported"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="audit_logs_`+time.Now().Format("2006-01-02")+`.csv"`)

	// Build query
	query := h.db.Model(&models.AuditLog{}).Preload("User")

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if startDate != "" {
		query = query.Where("timestamp >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("timestamp <= ?", endDate)
	}

	var auditLogs []models.AuditLog
	if err := query.Order("timestamp DESC").Find(&auditLogs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	// Generate CSV
	csvContent := "Timestamp,User,Action,Entity Type,Entity ID,IP Address,User Agent,Description\n"

	for _, log := range auditLogs {
		username := ""
		if log.User != nil {
			username = log.User.Username
		}

		description := log.Action
		if description == "" {
			description = "N/A"
		}

		ipAddress := log.IPAddress
		if ipAddress == "" {
			ipAddress = "N/A"
		}

		userAgent := log.UserAgent
		if userAgent == "" {
			userAgent = "N/A"
		}

		csvContent += fmt.Sprintf("%s,\"%s\",%s,%s,%s,\"%s\",\"%s\",\"%s\"\n",
			log.Timestamp.Format("2006-01-02 15:04:05"),
			username,
			log.Action,
			log.EntityType,
			log.EntityID,
			ipAddress,
			userAgent,
			description,
		)
	}

	c.String(http.StatusOK, csvContent)
}

// ================================================================
// PERMISSION MANAGEMENT
// ================================================================

// GetPermissions returns all available permissions
func (h *SecurityHandler) GetPermissions(c *gin.Context) {
	if !h.hasPermission(c, "permission.read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	permissions := map[string][]string{
		"roles": {
			"role.read",
			"role.create",
			"role.update",
			"role.delete",
		},
		"users": {
			"user.read",
			"user.create",
			"user.update",
			"user.delete",
			"user.assign_role",
			"user.revoke_role",
		},
		"jobs": {
			"job.read",
			"job.create",
			"job.update",
			"job.delete",
			"job.assign_device",
			"job.manage_templates",
		},
		"devices": {
			"device.read",
			"device.create",
			"device.update",
			"device.delete",
			"device.maintenance",
			"device.location",
		},
		"customers": {
			"customer.read",
			"customer.create",
			"customer.update",
			"customer.delete",
		},
		"financial": {
			"financial.read",
			"financial.create",
			"financial.update",
			"financial.delete",
			"financial.reports",
			"financial.invoices",
		},
		"documents": {
			"document.read",
			"document.create",
			"document.update",
			"document.delete",
			"document.sign",
		},
		"analytics": {
			"analytics.read",
			"analytics.export",
		},
		"system": {
			"audit.read",
			"permission.read",
			"system.admin",
		},
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

// CheckPermission checks if current user has a specific permission
func (h *SecurityHandler) CheckPermission(c *gin.Context) {
	permission := c.Query("permission")
	if permission == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission parameter is required"})
		return
	}

	hasPermission := h.hasPermission(c, permission)
	c.JSON(http.StatusOK, gin.H{"hasPermission": hasPermission})
}

// ================================================================
// HELPER FUNCTIONS
// ================================================================

// hasPermission checks if the current user has the specified permission
func (h *SecurityHandler) hasPermission(c *gin.Context, permission string) bool {
	currentUser, exists := GetCurrentUser(c)
	if !exists {
		return false
	}

	// System admin has all permissions
	if currentUser.Username == "admin" {
		return true
	}

	// Get user's active roles
	var userRoles []models.UserRole
	result := h.db.Preload("Role").Where("userID = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", 
		currentUser.UserID, true, time.Now()).Find(&userRoles)
	
	if result.Error != nil {
		return false
	}

	// Check if any role has the required permission
	for _, userRole := range userRoles {
		if userRole.Role == nil || !userRole.Role.IsActive {
			continue
		}

		var permissions []string
		if err := json.Unmarshal(userRole.Role.Permissions, &permissions); err != nil {
			continue
		}

		for _, perm := range permissions {
			if perm == permission || perm == "*" {
				return true
			}
		}
	}

	return false
}

// logAction logs an action to the audit trail
func (h *SecurityHandler) logAction(c *gin.Context, action, entityType, entityID string, oldValues, newValues interface{}) {
	currentUser, exists := GetCurrentUser(c)
	
	auditLog := models.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Timestamp:  time.Now(),
	}

	if exists {
		auditLog.UserID = &currentUser.UserID
		// Get session ID if available
		if sessionID, exists := c.Get("sessionID"); exists {
			if sid, ok := sessionID.(string); ok {
				auditLog.SessionID = sid
			}
		}
	}

	if oldValues != nil {
		if data, err := json.Marshal(oldValues); err == nil {
			auditLog.OldValues = data
		}
	}

	if newValues != nil {
		if data, err := json.Marshal(newValues); err == nil {
			auditLog.NewValues = data
		}
	}

	// Save audit log (ignore errors to not break the main operation)
	h.db.Create(&auditLog)
}

// InitializeDefaultRoles creates default system roles
func (h *SecurityHandler) InitializeDefaultRoles() error {
	// Admin role
	adminPermissions := []string{"*"} // All permissions
	adminPermsJSON, _ := json.Marshal(adminPermissions)
	
	adminRole := models.Role{
		Name:         "admin",
		DisplayName:  "Administrator",
		Description:  "Full system access",
		Permissions:  adminPermsJSON,
		IsSystemRole: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Manager role
	managerPermissions := []string{
		"job.read", "job.create", "job.update", "job.delete", "job.assign_device", "job.manage_templates",
		"device.read", "device.create", "device.update", "device.maintenance", "device.location",
		"customer.read", "customer.create", "customer.update",
		"financial.read", "financial.create", "financial.update", "financial.reports", "financial.invoices",
		"document.read", "document.create", "document.update", "document.sign",
		"analytics.read", "analytics.export",
		"user.read",
	}
	managerPermsJSON, _ := json.Marshal(managerPermissions)
	
	managerRole := models.Role{
		Name:         "manager",
		DisplayName:  "Manager",
		Description:  "Equipment and job management",
		Permissions:  managerPermsJSON,
		IsSystemRole: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Employee role
	employeePermissions := []string{
		"job.read", "job.update", "job.assign_device",
		"device.read", "device.update", "device.location",
		"customer.read",
		"document.read", "document.create",
		"analytics.read",
	}
	employeePermsJSON, _ := json.Marshal(employeePermissions)
	
	employeeRole := models.Role{
		Name:         "employee",
		DisplayName:  "Employee",
		Description:  "Basic operations access",
		Permissions:  employeePermsJSON,
		IsSystemRole: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Viewer role
	viewerPermissions := []string{
		"job.read",
		"device.read",
		"customer.read",
		"document.read",
		"analytics.read",
	}
	viewerPermsJSON, _ := json.Marshal(viewerPermissions)
	
	viewerRole := models.Role{
		Name:         "viewer",
		DisplayName:  "Viewer",
		Description:  "Read-only access",
		Permissions:  viewerPermsJSON,
		IsSystemRole: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create roles if they don't exist
	roles := []models.Role{adminRole, managerRole, employeeRole, viewerRole}
	for _, role := range roles {
		var existing models.Role
		result := h.db.Where("name = ?", role.Name).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := h.db.Create(&role).Error; err != nil {
				return fmt.Errorf("failed to create role %s: %v", role.Name, err)
			}
		}
	}

	return nil
}

// SecurityAuditPage displays the security audit page
func (h *SecurityHandler) SecurityAuditPage(c *gin.Context) {
	if !h.hasPermission(c, "audit.view") {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"error": "Access denied: Security audit requires appropriate permissions",
			"user":  currentUser,
		})
		return
	}

	currentUser, _ := GetCurrentUser(c)
	c.HTML(http.StatusOK, "security_audit.html", gin.H{
		"title":       "Security Audit",
		"user":        currentUser,
		"currentPage": "security",
	})
}