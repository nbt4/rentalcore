package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db     *gorm.DB
	config *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, config: cfg}
}

// LoginForm displays the login page
func (h *AuthHandler) LoginForm(c *gin.Context) {
	// Check if user is already logged in
	if sessionID, err := c.Cookie("session_id"); err == nil && sessionID != "" {
		if h.validateSession(sessionID) {
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login",
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var loginData struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.ShouldBind(&loginData); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"title": "Login",
			"error": "Please fill in all fields",
		})
		return
	}

	// Find user by username
	var user models.User
	if err := h.db.Where("username = ? AND is_active = ?", loginData.Username, true).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login",
			"error": "Invalid username or password",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginData.Password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login",
			"error": "Invalid username or password",
		})
		return
	}

	// Check if user has 2FA enabled
	var twoFAEnabled bool
	h.db.Raw("SELECT COALESCE(is_enabled, false) FROM user_2fa WHERE user_id = ?", user.UserID).Scan(&twoFAEnabled)

	if twoFAEnabled {
		// Store user info in session temporarily for 2FA verification
		tempSessionID := h.generateSessionID()
		tempSession := models.Session{
			SessionID: tempSessionID,
			UserID:    user.UserID,
			ExpiresAt: time.Now().Add(5 * time.Minute), // Short-lived for 2FA verification
			CreatedAt: time.Now(),
		}

		if err := h.db.Create(&tempSession).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{
				"title": "Login",
				"error": "Login failed. Please try again.",
			})
			return
		}

		// Redirect to 2FA verification page
		cookieDomain := getCookieDomain(c)
		c.SetCookie("temp_session_id", tempSessionID, 300, "/", cookieDomain, false, true) // 5 minutes
		c.Redirect(http.StatusSeeOther, "/login/2fa")
		return
	}

	// Create full session (no 2FA required)
	sessionID := h.generateSessionID()
	sessionTimeout := time.Duration(h.config.Security.SessionTimeout) * time.Second
	session := models.Session{
		SessionID: sessionID,
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(sessionTimeout),
		CreatedAt: time.Now(),
	}

	fmt.Printf("DEBUG: Creating session for user %s (ID: %d)\n", user.Username, user.UserID)
	if err := h.db.Create(&session).Error; err != nil {
		fmt.Printf("DEBUG: Session creation failed: %v\n", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"title": "Login",
			"error": "Login failed. Please try again.",
		})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	h.db.Save(&user)

	// Set cookie with shared domain for SSO
	cookieDomain := getCookieDomain(c)
	c.SetCookie("session_id", sessionID, h.config.Security.SessionTimeout, "/", cookieDomain, false, true)
	fmt.Printf("DEBUG: Login successful, session created: %s with cookie domain: %s\n", sessionID, cookieDomain)

	// Check if password change is required
	if user.ForcePasswordChange {
		fmt.Printf("DEBUG: User %s must change password on first login\n", user.Username)
		c.Redirect(http.StatusSeeOther, "/auth/force-password-change")
		return
	}

	// Redirect to home
	c.Redirect(http.StatusSeeOther, "/")
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	if sessionID, err := c.Cookie("session_id"); err == nil {
		// Delete session from database
		h.db.Where("session_id = ?", sessionID).Delete(&models.Session{})
	}

	// Clear cookie with same domain used for setting
	cookieDomain := getCookieDomain(c)
	c.SetCookie("session_id", "", -1, "/", cookieDomain, false, true)

	// Redirect to login
	c.Redirect(http.StatusSeeOther, "/login")
}

// ShowForcePasswordChange displays the forced password change form
func (h *AuthHandler) ShowForcePasswordChange(c *gin.Context) {
	// Ensure user is logged in
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Verify session
	var session models.Session
	if err := h.db.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, session.UserID).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	c.HTML(http.StatusOK, "force_password_change.html", gin.H{
		"title":    "Password Change Required",
		"username": user.Username,
	})
}

// HandleForcePasswordChange processes the forced password change
func (h *AuthHandler) HandleForcePasswordChange(c *gin.Context) {
	var changeData struct {
		CurrentPassword string `form:"current_password" binding:"required"`
		NewPassword     string `form:"new_password" binding:"required,min=8"`
		ConfirmPassword string `form:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBind(&changeData); err != nil {
		c.HTML(http.StatusBadRequest, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "Please fill out all fields. Password must be at least 8 characters.",
		})
		return
	}

	// Verify passwords match
	if changeData.NewPassword != changeData.ConfirmPassword {
		c.HTML(http.StatusBadRequest, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "New passwords do not match",
		})
		return
	}

	// Get session
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	var session models.Session
	if err := h.db.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, session.UserID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "An error occurred. Please try again.",
		})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(changeData.CurrentPassword)); err != nil {
		c.HTML(http.StatusUnauthorized, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "Current password is incorrect",
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(changeData.NewPassword), 14)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "An error occurred. Please try again.",
		})
		return
	}

	// Update user password and remove force flag
	user.PasswordHash = string(hashedPassword)
	user.ForcePasswordChange = false
	if err := h.db.Save(&user).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "force_password_change.html", gin.H{
			"title": "Password Change Required",
			"error": "An error occurred saving your password. Please try again.",
		})
		return
	}

	fmt.Printf("DEBUG: User %s successfully changed password on first login\n", user.Username)

	// Redirect to home with success message
	c.Redirect(http.StatusSeeOther, "/?password_changed=1")
}

// Login2FAForm shows the 2FA verification page
func (h *AuthHandler) Login2FAForm(c *gin.Context) {
	c.HTML(http.StatusOK, "login_2fa.html", gin.H{
		"title": "Two-Factor Authentication",
	})
}

// Login2FAVerify handles 2FA verification during login
func (h *AuthHandler) Login2FAVerify(c *gin.Context) {
	var verifyData struct {
		Code string `form:"code" binding:"required"`
	}

	if err := c.ShouldBind(&verifyData); err != nil {
		c.HTML(http.StatusBadRequest, "login_2fa.html", gin.H{
			"title": "Two-Factor Authentication",
			"error": "Please enter a verification code",
		})
		return
	}

	// Get temporary session
	tempSessionID, err := c.Cookie("temp_session_id")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Find temp session
	var tempSession models.Session
	if err := h.db.Where("session_id = ? AND expires_at > ?", tempSessionID, time.Now()).First(&tempSession).Error; err != nil {
		cookieDomain := getCookieDomain(c)
		c.SetCookie("temp_session_id", "", -1, "/", cookieDomain, false, true) // Clear cookie
		c.HTML(http.StatusUnauthorized, "login_2fa.html", gin.H{
			"title": "Two-Factor Authentication",
			"error": "Session expired. Please log in again.",
		})
		return
	}

	// Get user and 2FA info
	var user models.User
	if err := h.db.Where("userID = ?", tempSession.UserID).First(&user).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Get 2FA secret using raw SQL
	var secret string
	if err := h.db.Raw("SELECT secret FROM user_2fa WHERE user_id = ? AND is_enabled = 1", user.UserID).Scan(&secret).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login_2fa.html", gin.H{
			"title": "Two-Factor Authentication",
			"error": "2FA not properly configured",
		})
		return
	}

	// Verify TOTP code
	valid := totp.Validate(verifyData.Code, secret)
	if !valid {
		// Check backup codes
		var backupCodesJSON string
		h.db.Raw("SELECT backup_codes FROM user_2fa WHERE user_id = ?", user.UserID).Scan(&backupCodesJSON)

		if backupCodesJSON != "" {
			var backupCodes []string
			if json.Unmarshal([]byte(backupCodesJSON), &backupCodes) == nil {
				for i, backupCode := range backupCodes {
					if backupCode == verifyData.Code {
						valid = true
						// Remove used backup code
						backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)
						newBackupCodesJSON, _ := json.Marshal(backupCodes)
						h.db.Exec("UPDATE user_2fa SET backup_codes = ? WHERE user_id = ?", string(newBackupCodesJSON), user.UserID)
						break
					}
				}
			}
		}
	}

	if !valid {
		c.HTML(http.StatusUnauthorized, "login_2fa.html", gin.H{
			"title": "Two-Factor Authentication",
			"error": "Invalid verification code",
		})
		return
	}

	// Delete temporary session
	cookieDomain := getCookieDomain(c)
	h.db.Delete(&tempSession)
	c.SetCookie("temp_session_id", "", -1, "/", cookieDomain, false, true)

	// Create full session
	sessionID := h.generateSessionID()
	sessionTimeout := time.Duration(h.config.Security.SessionTimeout) * time.Second
	session := models.Session{
		SessionID: sessionID,
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(sessionTimeout),
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login_2fa.html", gin.H{
			"title": "Two-Factor Authentication",
			"error": "Login failed. Please try again.",
		})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	h.db.Save(&user)

	// Set cookie with shared domain for SSO
	c.SetCookie("session_id", sessionID, h.config.Security.SessionTimeout, "/", cookieDomain, false, true)

	// Redirect to home
	c.Redirect(http.StatusSeeOther, "/")
}

// AuthMiddleware checks if user is authenticated
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Helper to check if request expects JSON or is API
		isAPI := strings.HasPrefix(c.Request.URL.Path, "/api") ||
			c.GetHeader("Accept") == "application/json" ||
			c.ContentType() == "application/json"

		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			if isAPI {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "code": "NO_SESSION"})
			} else {
				log.Printf("DEBUG: AuthMiddleware: No session cookie found for %s, redirecting to /login", c.Request.URL.Path)
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
			}
			return
		}

		// Validate session
		var session models.Session
		if err := h.db.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
			log.Printf("DEBUG: AuthMiddleware: Session validation failed for %s: %v", sessionID, err)
			// Clean up invalid session cookie
			cookieDomain := getCookieDomain(c)
			c.SetCookie("session_id", "", -1, "/", cookieDomain, false, true)
			
			if isAPI {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired", "code": "SESSION_EXPIRED"})
			} else {
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
			}
			return
		}

		// Load the user and verify they are still active
		var user models.User
		if err := h.db.Where("userID = ? AND is_active = ?", session.UserID, true).First(&user).Error; err != nil {
			log.Printf("DEBUG: AuthMiddleware: User not found or inactive for session %s (UserID: %d): %v", sessionID, session.UserID, err)
			// Delete the session since user is inactive/deleted
			cookieDomain := getCookieDomain(c)
			h.db.Where("session_id = ?", sessionID).Delete(&models.Session{})
			c.SetCookie("session_id", "", -1, "/", cookieDomain, false, true)
			
			if isAPI {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User inactive", "code": "USER_INACTIVE"})
			} else {
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
			}
			return
		}

		// Store user in context (as pointer for downstream middlewares)
		c.Set("user", &user)
		c.Set("userid", session.UserID)
		c.Next()
	}
}

// validateSession checks if a session is valid and the user is active
func (h *AuthHandler) validateSession(sessionID string) bool {
	var session models.Session
	if err := h.db.Where("session_id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		return false
	}

	// Also check if the user is still active
	var user models.User
	return h.db.Where("userID = ? AND is_active = ?", session.UserID, true).First(&user).Error == nil
}

// generateSessionID creates a new session ID
func (h *AuthHandler) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getCookieDomain determines the appropriate cookie domain for SSO
// Returns empty string for localhost, otherwise returns the parent domain with leading dot
func getCookieDomain(c *gin.Context) string {
	// Check if COOKIE_DOMAIN is set in environment (highest priority for explicit control)
	if domain := os.Getenv("COOKIE_DOMAIN"); domain != "" {
		return domain
	}

	host := c.Request.Host

	// Remove port if present
	if idx := len(host) - 1; idx >= 0 {
		for i := len(host) - 1; i >= 0; i-- {
			if host[i] == ':' {
				host = host[:i]
				break
			}
		}
	}

	// Localhost: no domain restriction
	if host == "localhost" || host == "127.0.0.1" {
		return ""
	}

	// For production domains like rent.server-nt.de, storage.server-nt.de
	// Extract parent domain: server-nt.de
	// Split by dots to get domain parts
	parts := []string{}
	for _, part := range []byte(host) {
		if part == '.' {
			parts = append(parts, ".")
		}
	}

	// Use simpler string splitting
	domainParts := []string{}
	temp := ""
	for i := 0; i < len(host); i++ {
		if host[i] == '.' {
			if temp != "" {
				domainParts = append(domainParts, temp)
				temp = ""
			}
		} else {
			temp += string(host[i])
		}
	}
	if temp != "" {
		domainParts = append(domainParts, temp)
	}

	fmt.Printf("DEBUG: getCookieDomain: host=%s, parsed parts=%v\n", host, domainParts)

	// If we have at least 3 parts (e.g., rent.server-nt.de), use parent domain
	// For rent.server-nt.de -> parts=["rent", "server-nt", "de"] -> return ".server-nt.de"
	if len(domainParts) >= 3 {
		parentDomain := domainParts[len(domainParts)-2] + "." + domainParts[len(domainParts)-1]
		fmt.Printf("DEBUG: getCookieDomain: returning parent domain: .%s\n", parentDomain)
		return "." + parentDomain // Leading dot for all subdomains
	}

	// If we have 2 parts (e.g., server-nt.de), also use it with leading dot
	if len(domainParts) == 2 {
		parentDomain := domainParts[0] + "." + domainParts[1]
		fmt.Printf("DEBUG: getCookieDomain: returning domain with dot: .%s\n", parentDomain)
		return "." + parentDomain
	}

	// Fallback: no domain restriction
	fmt.Printf("DEBUG: getCookieDomain: fallback to empty string\n")
	return ""
}

// CleanupExpiredSessions removes expired sessions from the database
func (h *AuthHandler) CleanupExpiredSessions() error {
	result := h.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		fmt.Printf("DEBUG: Cleaned up %d expired sessions\n", result.RowsAffected)
	}

	return nil
}

// StartSessionCleanup starts a background goroutine to periodically clean up expired sessions
func (h *AuthHandler) StartSessionCleanup() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute) // Clean up every 30 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := h.CleanupExpiredSessions(); err != nil {
					fmt.Printf("ERROR: Failed to cleanup expired sessions: %v\n", err)
				}
			}
		}
	}()
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CreateUser creates a new user (helper function for user management)
func (h *AuthHandler) CreateUser(username, email, password, firstName, lastName string) error {
	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("username = ? OR email = ?", username, email).First(&existingUser).Error; err == nil {
		return gorm.ErrDuplicatedKey
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	// Create user
	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return h.db.Create(&user).Error
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	if user, exists := c.Get("user"); exists {
		if uPtr, ok := user.(*models.User); ok && uPtr != nil {
			return uPtr, true
		}
		if u, ok := user.(models.User); ok {
			return &u, true
		}
	}
	return nil, false
}

// GetAppDomains returns the cross-navigation domains from context
func GetAppDomains(c *gin.Context) (string, string) {
	storageCoreDomain := ""
	rentalCoreDomain := ""

	if val, exists := c.Get("WarehouseCoreDomain"); exists {
		if domain, ok := val.(string); ok {
			storageCoreDomain = domain
		}
	}

	if val, exists := c.Get("RentalCoreDomain"); exists {
		if domain, ok := val.(string); ok {
			rentalCoreDomain = domain
		}
	}

	return storageCoreDomain, rentalCoreDomain
}

// User Management Web Interface Handlers

// ListUsers displays all users
func (h *AuthHandler) ListUsers(c *gin.Context) {
	fmt.Printf("DEBUG: ListUsers called - URL: %s\n", c.Request.URL.Path)

	var users []models.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		fmt.Printf("DEBUG: Database error: %v\n", err)
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": currentUser})
		return
	}

	fmt.Printf("DEBUG: Found %d users\n", len(users))
	currentUser, exists := GetCurrentUser(c)
	fmt.Printf("DEBUG: Current user exists: %v, User: %+v\n", exists, currentUser)

	fmt.Printf("DEBUG: Rendering users_list.html with currentPage = 'users'\n")
	c.HTML(http.StatusOK, "users_list.html", gin.H{
		"title":       "User Management",
		"users":       users,
		"user":        currentUser,
		"currentPage": "users",
	})
	fmt.Printf("DEBUG: ListUsers template rendered\n")
}

// NewUserForm displays the create user form
func (h *AuthHandler) NewUserForm(c *gin.Context) {
	// Debug: Let's see what's happening
	fmt.Printf("DEBUG: NewUserForm called - URL: %s\n", c.Request.URL.Path)

	currentUser, exists := GetCurrentUser(c)
	fmt.Printf("DEBUG: User exists: %v, User: %+v\n", exists, currentUser)

	if !exists || currentUser == nil {
		fmt.Printf("DEBUG: No user found, redirecting to login\n")
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	fmt.Printf("DEBUG: Rendering user_form.html template\n")
	c.HTML(http.StatusOK, "user_form.html", gin.H{
		"title":    "Create New User",
		"formUser": &models.User{},
		"user":     currentUser,
	})
	fmt.Printf("DEBUG: Template rendered successfully\n")
}

// CreateUserWeb handles user creation from web form
func (h *AuthHandler) CreateUserWeb(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	isActiveStr := c.PostForm("is_active")

	isActive := isActiveStr == "on" || isActiveStr == "true"

	if username == "" || email == "" || password == "" {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusBadRequest, "user_form.html", gin.H{
			"title": "Create New User",
			"formUser": &models.User{
				Username:  username,
				Email:     email,
				FirstName: firstName,
				LastName:  lastName,
				IsActive:  isActive,
			},
			"user":  currentUser,
			"error": "Username, email and password are required",
		})
		return
	}

	if err := h.CreateUser(username, email, password, firstName, lastName); err != nil {
		var errorMsg string
		if err == gorm.ErrDuplicatedKey {
			errorMsg = "User with this username or email already exists"
		} else {
			errorMsg = err.Error()
		}

		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "user_form.html", gin.H{
			"title": "Create New User",
			"formUser": &models.User{
				Username:  username,
				Email:     email,
				FirstName: firstName,
				LastName:  lastName,
				IsActive:  isActive,
			},
			"user":  currentUser,
			"error": errorMsg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// GetUser displays user details
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.Where("userID = ?", userID).First(&user).Error; err != nil {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "User not found", "user": currentUser})
		return
	}

	currentUser, _ := GetCurrentUser(c)
	c.HTML(http.StatusOK, "user_detail.html", gin.H{
		"title":    "User Details",
		"viewUser": user,
		"user":     currentUser,
	})
}

// EditUserForm displays the edit user form
func (h *AuthHandler) EditUserForm(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.Where("userID = ?", userID).First(&user).Error; err != nil {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "User not found", "user": currentUser})
		return
	}

	currentUser, _ := GetCurrentUser(c)
	c.HTML(http.StatusOK, "user_form.html", gin.H{
		"title":    "Edit User",
		"formUser": user,
		"user":     currentUser,
	})
}

// UpdateUser handles user updates
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.Where("userID = ?", userID).First(&user).Error; err != nil {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "User not found", "user": currentUser})
		return
	}

	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	isActiveStr := c.PostForm("is_active")

	isActive := isActiveStr == "on" || isActiveStr == "true"

	if username == "" || email == "" {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusBadRequest, "user_form.html", gin.H{
			"title":    "Edit User",
			"formUser": user,
			"user":     currentUser,
			"error":    "Username and email are required",
		})
		return
	}

	// Check for duplicate username/email (excluding current user)
	var existingUser models.User
	if err := h.db.Where("(username = ? OR email = ?) AND userID != ?", username, email, userID).First(&existingUser).Error; err == nil {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusBadRequest, "user_form.html", gin.H{
			"title":    "Edit User",
			"formUser": user,
			"user":     currentUser,
			"error":    "User with this username or email already exists",
		})
		return
	}

	// Update user fields
	user.Username = username
	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName
	user.IsActive = isActive
	user.UpdatedAt = time.Now()

	// Update password if provided
	if password != "" {
		hashedPassword, err := HashPassword(password)
		if err != nil {
			currentUser, _ := GetCurrentUser(c)
			c.HTML(http.StatusInternalServerError, "user_form.html", gin.H{
				"title":    "Edit User",
				"formUser": user,
				"user":     currentUser,
				"error":    "Failed to hash password",
			})
			return
		}
		user.PasswordHash = hashedPassword
	}

	if err := h.db.Save(&user).Error; err != nil {
		currentUser, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "user_form.html", gin.H{
			"title":    "Edit User",
			"formUser": user,
			"user":     currentUser,
			"error":    err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

// DeleteUser handles user deletion
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Don't allow deleting the current user
	currentUser, exists := GetCurrentUser(c)
	if exists && currentUser.UserID == parseUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	if err := h.db.Where("userID = ?", userID).Delete(&models.User{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ListUsersAPI returns users in JSON format for API calls
func (h *AuthHandler) ListUsersAPI(c *gin.Context) {
	var users []models.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	// Remove sensitive information (passwords) before returning
	for i := range users {
		users[i].PasswordHash = "" // Don't expose password hashes
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
	})
}

// Helper function to parse user ID
func parseUserID(userIDStr string) uint {
	if userIDStr == "" {
		return 0
	}

	// Convert string to uint
	if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
		return uint(id)
	}

	return 0
}

// ================================================================
// ADMIN USER MANAGEMENT FUNCTIONS
// ================================================================

// AdminSetUserPassword allows admins to set passwords for other users
func (h *AuthHandler) AdminSetUserPassword(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if current user has permission to manage user passwords
	if !h.hasPasswordResetPermission(currentUser) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	userID := c.Param("id")
	var request struct {
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the target user
	var targetUser models.User
	if err := h.db.Where("userID = ?", userID).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Hash the new password
	hashedPassword, err := HashPassword(request.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update the user's password and enforce change on next login
	targetUser.PasswordHash = hashedPassword
	targetUser.ForcePasswordChange = true
	targetUser.UpdatedAt = time.Now()

	if err := h.db.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Log the action
	h.logAdminAction(c, "set_password", "user", userID, currentUser.UserID)

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// AdminBlockUser allows admins to block/unblock user logins
func (h *AuthHandler) AdminBlockUser(c *gin.Context) {
	currentUser, exists := GetCurrentUser(c)
	if !exists || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if current user has admin privileges
	if !h.hasAdminPermission(currentUser) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

	userID := c.Param("id")
	var request struct {
		IsActive bool `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Don't allow blocking self
	if fmt.Sprintf("%d", currentUser.UserID) == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block your own account"})
		return
	}

	// Find the target user
	var targetUser models.User
	if err := h.db.Where("userID = ?", userID).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	oldStatus := targetUser.IsActive
	targetUser.IsActive = request.IsActive
	targetUser.UpdatedAt = time.Now()

	if err := h.db.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	// If user is being blocked, invalidate all their sessions
	if !request.IsActive {
		h.db.Where("user_id = ?", userID).Delete(&models.Session{})
	}

	// Log the action
	action := "unblock_user"
	if !request.IsActive {
		action = "block_user"
	}
	h.logAdminAction(c, action, "user", userID, currentUser.UserID)

	statusText := "unblocked"
	if !request.IsActive {
		statusText = "blocked"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("User %s successfully", statusText),
		"oldStatus": oldStatus,
		"newStatus": request.IsActive,
	})
}

func (h *AuthHandler) hasPasswordResetPermission(user *models.User) bool {
	if user == nil {
		return false
	}

	if h.hasAdminPermission(user) {
		return true
	}

	var userRoles []models.UserRole
	if err := h.db.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
		return false
	}

	for _, userRole := range userRoles {
		if userRole.Role == nil || !userRole.Role.IsActive {
			continue
		}

		roleName := strings.ToLower(userRole.Role.Name)
		if roleName == "manager" || roleName == "users_manager" {
			return true
		}

		var permissions []string
		if err := json.Unmarshal(userRole.Role.Permissions, &permissions); err != nil {
			continue
		}

		for _, perm := range permissions {
			if perm == "users.manage" || perm == "users.edit" {
				return true
			}
		}
	}

	return false
}

// hasAdminPermission checks if user has admin privileges
func (h *AuthHandler) hasAdminPermission(user *models.User) bool {
	// System admin always has permission
	if user.Username == "admin" {
		return true
	}

	// Check if user has admin role or specific permissions
	var userRoles []models.UserRole
	if err := h.db.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
		return false
	}

	for _, userRole := range userRoles {
		if userRole.Role == nil || !userRole.Role.IsActive {
			continue
		}

		var permissions []string
		if err := json.Unmarshal(userRole.Role.Permissions, &permissions); err != nil {
			continue
		}

		for _, perm := range permissions {
			if perm == "*" || perm == "users.manage" || perm == "users.admin" {
				return true
			}
		}
	}

	return false
}

// LoginAPI handles JSON login requests from the React frontend.
func (h *AuthHandler) LoginAPI(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Benutzername und Passwort erforderlich"})
		return
	}

	var user models.User
	if err := h.db.Where("username = ? AND is_active = ?", req.Username, true).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Ungültige Zugangsdaten"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Ungültige Zugangsdaten"})
		return
	}

	sessionID := h.generateSessionID()
	sessionTimeout := time.Duration(h.config.Security.SessionTimeout) * time.Second
	session := models.Session{
		SessionID: sessionID,
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(sessionTimeout),
		CreatedAt: time.Now(),
	}
	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Sitzung konnte nicht erstellt werden"})
		return
	}

	now := time.Now()
	user.LastLogin = &now
	h.db.Save(&user)

	cookieDomain := getCookieDomain(c)
	c.SetCookie("session_id", sessionID, h.config.Security.SessionTimeout, "/", cookieDomain, false, true)

	var userRoles []models.UserRole
	h.db.Preload("Role").Where("user_id = ?", user.UserID).Find(&userRoles)

	type roleResp struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	roleList := make([]roleResp, 0, len(userRoles))
	for _, ur := range userRoles {
		if ur.Role.Name != "" {
			roleList = append(roleList, roleResp{ID: ur.RoleID, Name: ur.Role.Name})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"UserID":                user.UserID,
			"Username":              user.Username,
			"Email":                 user.Email,
			"FirstName":             user.FirstName,
			"LastName":              user.LastName,
			"IsActive":              user.IsActive,
			"Roles":                 roleList,
			"force_password_change": user.ForcePasswordChange,
		},
		"force_password_change": user.ForcePasswordChange,
	})
}

// LogoutAPI handles JSON logout requests from the React frontend.
func (h *AuthHandler) LogoutAPI(c *gin.Context) {
	if sessionID, err := c.Cookie("session_id"); err == nil {
		h.db.Where("session_id = ?", sessionID).Delete(&models.Session{})
	}
	cookieDomain := getCookieDomain(c)
	c.SetCookie("session_id", "", -1, "/", cookieDomain, false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MeAPI returns the currently authenticated user as JSON.
func (h *AuthHandler) MeAPI(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nicht authentifiziert"})
		return
	}

	var userRoles []models.UserRole
	h.db.Preload("Role").Where("user_id = ?", user.UserID).Find(&userRoles)

	type roleResp struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	roleList := make([]roleResp, 0, len(userRoles))
	for _, ur := range userRoles {
		if ur.Role.Name != "" {
			roleList = append(roleList, roleResp{ID: ur.RoleID, Name: ur.Role.Name})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"UserID":                user.UserID,
		"Username":              user.Username,
		"Email":                 user.Email,
		"FirstName":             user.FirstName,
		"LastName":              user.LastName,
		"IsActive":              user.IsActive,
		"Roles":                 roleList,
		"force_password_change": user.ForcePasswordChange,
	})
}

// ChangePasswordAPI handles JSON password change requests from the React frontend.
func (h *AuthHandler) ChangePasswordAPI(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nicht authentifiziert"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Felder fehlen"})
		return
	}

	var fullUser models.User
	if err := h.db.First(&fullUser, user.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Benutzer nicht gefunden"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(fullUser.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Aktuelles Passwort ist falsch"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Passwort konnte nicht gehasht werden"})
		return
	}

	if err := h.db.Model(&fullUser).Updates(map[string]interface{}{
		"password_hash":         string(hash),
		"force_password_change": false,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Passwort konnte nicht gespeichert werden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Passwort erfolgreich geändert"})
}

// logAdminAction logs admin actions for auditing
func (h *AuthHandler) logAdminAction(c *gin.Context, action, entityType, entityID string, adminUserID uint) {
	auditLog := models.AuditLog{
		UserID:     &adminUserID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		Timestamp:  time.Now(),
	}

	// Save audit log (ignore errors to not break main operation)
	h.db.Create(&auditLog)
}
