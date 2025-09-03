package handlers

import (
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	db           *gorm.DB
	config       *config.Config
	webauthnHandler *WebAuthnHandler
}

func NewProfileHandler(db *gorm.DB, cfg *config.Config) *ProfileHandler {
	return &ProfileHandler{
		db:           db,
		config:       cfg,
		webauthnHandler: NewWebAuthnHandler(db, cfg),
	}
}

// ProfileSettingsForm displays the comprehensive profile settings page with security features
func (h *ProfileHandler) ProfileSettingsForm(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists || currentUser == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get or create user preferences
	var preferences models.UserPreferences
	if err := h.db.Where("user_id = ?", currentUser.UserID).First(&preferences).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			preferences = models.UserPreferences{
				UserID:                   currentUser.UserID,
				Language:                 "de",
				Theme:                    "dark",
				TimeZone:                 "Europe/Berlin",
				DateFormat:               "DD.MM.YYYY",
				TimeFormat:               "24h",
				EmailNotifications:       true,
				SystemNotifications:      true,
				JobStatusNotifications:   true,
				DeviceAlertNotifications: true,
				ItemsPerPage:             25,
				DefaultView:              "list",
				ShowAdvancedOptions:      false,
				AutoSaveEnabled:          true,
				CreatedAt:                time.Now(),
				UpdatedAt:                time.Now(),
			}
			if err := h.db.Create(&preferences).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{
					"error": "Failed to create user preferences",
					"user":  currentUser,
				})
				return
			}
		} else {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": "Failed to load user preferences",
				"user":  currentUser,
			})
			return
		}
	}

	// Get 2FA status using raw SQL to avoid JSON scanning issues
	var twoFAEnabled bool
	h.db.Raw("SELECT COALESCE(is_enabled, false) FROM user_2fa WHERE user_id = ?", currentUser.UserID).Scan(&twoFAEnabled)

	// Get passkeys
	var passkeys []models.UserPasskey
	h.db.Where("user_id = ? AND is_active = ?", currentUser.UserID, true).Find(&passkeys)
	
	// Remove sensitive data from passkeys before sending to template
	for i := range passkeys {
		passkeys[i].PublicKey = nil
	}

	// Get recent authentication attempts for security overview
	var recentAttempts []models.AuthenticationAttempt
	h.db.Where("user_id = ?", currentUser.UserID).
		Order("attempted_at DESC").
		Limit(5).
		Find(&recentAttempts)

	c.HTML(http.StatusOK, "profile_settings_standalone.html", gin.H{
		"title":           "Profile Settings",
		"user":            currentUser,
		"preferences":     preferences,
		"twoFAEnabled":    twoFAEnabled,
		"passkeys":        passkeys,
		"recentAttempts":  recentAttempts,
		"currentPage":     "profile",
	})
}

// UpdateProfileSettings handles profile settings updates
func (h *ProfileHandler) UpdateProfileSettings(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse form data for different sections
	section := c.PostForm("section")

	switch section {
	case "personal":
		h.updatePersonalInfo(c, currentUser)
	case "preferences":
		h.updatePreferences(c, currentUser)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section"})
	}
}

// updatePersonalInfo updates user's personal information
func (h *ProfileHandler) updatePersonalInfo(c *gin.Context, currentUser *models.User) {
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Update user fields
	currentUser.FirstName = firstName
	currentUser.LastName = lastName
	currentUser.Email = email
	currentUser.UpdatedAt = time.Now()

	// Update password if provided
	if password != "" {
		hashedPassword, err := HashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		currentUser.PasswordHash = hashedPassword
	}

	// Save user
	if err := h.db.Save(currentUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user information"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Personal information updated successfully"})
}

// updatePreferences updates user preferences
func (h *ProfileHandler) updatePreferences(c *gin.Context, currentUser *models.User) {
	var preferences models.UserPreferences
	if err := h.db.Where("user_id = ?", currentUser.UserID).First(&preferences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load preferences"})
		return
	}

	// Update preferences from form data
	preferences.Language = c.PostForm("language")
	preferences.Theme = c.PostForm("theme")
	preferences.TimeZone = c.PostForm("time_zone")
	preferences.DateFormat = c.PostForm("date_format")
	preferences.TimeFormat = c.PostForm("time_format")
	
	// Parse boolean values
	preferences.EmailNotifications = c.PostForm("email_notifications") == "on"
	preferences.SystemNotifications = c.PostForm("system_notifications") == "on"
	preferences.JobStatusNotifications = c.PostForm("job_status_notifications") == "on"
	preferences.DeviceAlertNotifications = c.PostForm("device_alert_notifications") == "on"
	preferences.ShowAdvancedOptions = c.PostForm("show_advanced_options") == "on"
	preferences.AutoSaveEnabled = c.PostForm("auto_save_enabled") == "on"
	
	// Parse integer values
	if itemsPerPage, err := strconv.Atoi(c.PostForm("items_per_page")); err == nil && itemsPerPage > 0 {
		preferences.ItemsPerPage = itemsPerPage
	}
	
	preferences.DefaultView = c.PostForm("default_view")
	preferences.UpdatedAt = time.Now()

	// Save preferences
	if err := h.db.Save(&preferences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// ================================================================
// 2FA ENDPOINTS (delegating to WebAuthnHandler)
// ================================================================

func (h *ProfileHandler) Setup2FA(c *gin.Context) {
	h.webauthnHandler.Setup2FA(c)
}

func (h *ProfileHandler) Verify2FA(c *gin.Context) {
	h.webauthnHandler.Verify2FA(c)
}

func (h *ProfileHandler) Disable2FA(c *gin.Context) {
	h.webauthnHandler.Disable2FA(c)
}

func (h *ProfileHandler) Get2FAStatus(c *gin.Context) {
	h.webauthnHandler.Get2FAStatus(c)
}

// ================================================================
// PASSKEY ENDPOINTS (delegating to WebAuthnHandler)
// ================================================================

func (h *ProfileHandler) StartPasskeyRegistration(c *gin.Context) {
	h.webauthnHandler.StartPasskeyRegistration(c)
}

func (h *ProfileHandler) CompletePasskeyRegistration(c *gin.Context) {
	h.webauthnHandler.CompletePasskeyRegistration(c)
}

func (h *ProfileHandler) ListUserPasskeys(c *gin.Context) {
	h.webauthnHandler.ListUserPasskeys(c)
}

func (h *ProfileHandler) DeletePasskey(c *gin.Context) {
	h.webauthnHandler.DeletePasskey(c)
}

// SecurityStatus returns the current security status for the user
func (h *ProfileHandler) SecurityStatus(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get 2FA status
	var twoFAEnabled bool
	h.db.Raw("SELECT COALESCE(is_enabled, false) FROM user_2fa WHERE user_id = ?", currentUser.UserID).Scan(&twoFAEnabled)

	// Get passkey count
	var passkeyCount int64
	h.db.Model(&models.UserPasskey{}).Where("user_id = ? AND is_active = ?", currentUser.UserID, true).Count(&passkeyCount)

	c.JSON(http.StatusOK, gin.H{
		"twoFAEnabled": twoFAEnabled,
		"passkeyCount": passkeyCount,
	})
}

// ================================================================
// HELPER FUNCTIONS
// ================================================================

