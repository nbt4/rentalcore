package middleware

import (
	"encoding/json"
	"net/http"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RBACMiddleware provides role-based access control middleware
type RBACMiddleware struct {
	db *gorm.DB
}

// NewRBACMiddleware creates a new RBAC middleware instance
func NewRBACMiddleware(db *gorm.DB) *RBACMiddleware {
	return &RBACMiddleware{db: db}
}

// getCurrentUser retrieves the current user from the Gin context
func getCurrentUser(c *gin.Context) (*models.User, bool) {
	userVal, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	if user, ok := userVal.(*models.User); ok && user != nil {
		return user, true
	}
	if user, ok := userVal.(models.User); ok {
		return &user, true
	}
	return nil, false
}

// RequireRole middleware ensures user has one of the required roles
func (m *RBACMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := getCurrentUser(c)
		if !exists || currentUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		if m.hasAnyRole(currentUser, roles) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// RequirePermission middleware ensures user has a specific permission
func (m *RBACMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, exists := getCurrentUser(c)
		if !exists || currentUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check if user has the required permission
		if m.hasPermission(currentUser, permission) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// RequireAdmin middleware ensures user has admin role
func (m *RBACMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole("admin")
}

// RequireAdminOrManager middleware ensures user has admin or manager role
func (m *RBACMiddleware) RequireAdminOrManager() gin.HandlerFunc {
	return m.RequireRole("admin", "manager")
}

// hasAnyRole checks if user has any of the specified roles
func (m *RBACMiddleware) hasAnyRole(user *models.User, roleNames []string) bool {
	// System admin always has access
	if user.IsAdmin || user.Username == "admin" {
		return true
	}

	// Get user roles from database
	var userRoles []models.UserRole
	if err := m.db.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
		return false
	}

	// Check if user has any of the required roles
	for _, userRole := range userRoles {
		if userRole.Role == nil || !userRole.Role.IsActive {
			continue
		}

		for _, requiredRole := range roleNames {
			if userRole.Role.Name == requiredRole {
				return true
			}
		}
	}

	return false
}

// hasPermission checks if user has a specific permission
func (m *RBACMiddleware) hasPermission(user *models.User, permission string) bool {
	// System admin always has all permissions
	if user.IsAdmin || user.Username == "admin" {
		return true
	}

	// Get user roles from database
	var userRoles []models.UserRole
	if err := m.db.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
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
			// Wildcard permission grants everything
			if perm == "*" {
				return true
			}
			// Exact match or prefix match (e.g., "job.*" matches "job.read")
			if perm == permission {
				return true
			}
			// Check for wildcard patterns
			if len(perm) > 0 && perm[len(perm)-1] == '*' {
				prefix := perm[:len(perm)-1]
				if len(permission) >= len(prefix) && permission[:len(prefix)] == prefix {
					return true
				}
			}
		}
	}

	return false
}

// GetUserRoles returns all active roles for a user
func (m *RBACMiddleware) GetUserRoles(user *models.User) []models.Role {
	if user.IsAdmin {
		return []models.Role{{Name: "admin", DisplayName: "Administrator", IsActive: true}}
	}

	var userRoles []models.UserRole
	if err := m.db.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
		return []models.Role{}
	}

	roles := make([]models.Role, 0, len(userRoles))
	for _, userRole := range userRoles {
		if userRole.Role != nil && userRole.Role.IsActive {
			roles = append(roles, *userRole.Role)
		}
	}

	return roles
}

// GetUserPermissions returns all permissions for a user
func (m *RBACMiddleware) GetUserPermissions(user *models.User) []string {
	roles := m.GetUserRoles(user)
	permissionSet := make(map[string]bool)

	for _, role := range roles {
		var permissions []string
		if err := json.Unmarshal(role.Permissions, &permissions); err != nil {
			continue
		}

		for _, perm := range permissions {
			permissionSet[perm] = true
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions
}
