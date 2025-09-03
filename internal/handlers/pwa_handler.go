package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PWAHandler struct {
	db *gorm.DB
}

func NewPWAHandler(db *gorm.DB) *PWAHandler {
	return &PWAHandler{db: db}
}

// Push notification subscription structure
type PushSubscription struct {
	Endpoint       string `json:"endpoint"`
	ExpirationTime *int64 `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// SubscribePush handles push notification subscription
func (h *PWAHandler) SubscribePush(c *gin.Context) {
	var subscription PushSubscription
	if err := c.ShouldBindJSON(&subscription); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert subscription to JSON (for logging if needed)
	_, _ = json.Marshal(subscription)

	// Save or update push subscription
	pushSub := models.PushSubscription{
		UserID:     currentUser.UserID,
		Endpoint:   subscription.Endpoint,
		KeysP256dh: subscription.Keys.P256dh,
		KeysAuth:   subscription.Keys.Auth,
		IsActive:   true,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	// Try to find existing subscription first
	var existing models.PushSubscription
	result := h.db.Where("userID = ? AND endpoint = ?", currentUser.UserID, subscription.Endpoint).First(&existing)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Create new subscription
		if err := h.db.Create(&pushSub).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save subscription"})
			return
		}
	} else {
		// Update existing subscription
		existing.KeysP256dh = subscription.Keys.P256dh
		existing.KeysAuth = subscription.Keys.Auth
		existing.IsActive = true
		existing.LastUsed = time.Now()
		
		if err := h.db.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push subscription saved successfully"})
}

// UnsubscribePush handles push notification unsubscription
func (h *PWAHandler) UnsubscribePush(c *gin.Context) {
	var request struct {
		Endpoint string `json:"endpoint"`
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

	// Deactivate subscription
	result := h.db.Model(&models.PushSubscription{}).
		Where("userID = ? AND endpoint = ?", currentUser.UserID, request.Endpoint).
		Update("is_active", false)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push subscription removed successfully"})
}

// SyncOfflineData handles offline data synchronization
func (h *PWAHandler) SyncOfflineData(c *gin.Context) {
	var request struct {
		Actions []models.OfflineSyncQueue `json:"actions"`
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

	results := make([]map[string]interface{}, len(request.Actions))
	
	for i, action := range request.Actions {
		result := h.processOfflineAction(currentUser.UserID, action)
		results[i] = result
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offline data synced",
		"results": results,
	})
}

// processOfflineAction processes individual offline actions
func (h *PWAHandler) processOfflineAction(userID uint, action models.OfflineSyncQueue) map[string]interface{} {
	result := map[string]interface{}{
		"id":     action.QueueID,
		"status": "error",
		"error":  "Unknown action type",
	}

	// Set user ID and timestamps
	action.UserID = userID
	action.Timestamp = time.Now()
	syncedAt := time.Now()
	action.SyncedAt = &syncedAt

	switch action.Action {
	case "create_job":
		if err := h.processCreateJob(action); err != nil {
			result["error"] = err.Error()
		} else {
			result["status"] = "success"
			delete(result, "error")
		}
	
	case "assign_device":
		if err := h.processAssignDevice(action); err != nil {
			result["error"] = err.Error()
		} else {
			result["status"] = "success"
			delete(result, "error")
		}
	
	case "update_status":
		if err := h.processUpdateStatus(action); err != nil {
			result["error"] = err.Error()
		} else {
			result["status"] = "success"
			delete(result, "error")
		}
	
	default:
		// Save unknown action for manual review
		h.db.Create(&action)
	}

	return result
}

// processCreateJob handles offline job creation
func (h *PWAHandler) processCreateJob(action models.OfflineSyncQueue) error {
	var jobData map[string]interface{}
	if err := json.Unmarshal(action.EntityData, &jobData); err != nil {
		return err
	}

	// Create job from offline data
	customerID := uint(jobData["customerID"].(float64))
	statusID := uint(jobData["statusID"].(float64))
	jobCategoryID := uint(jobData["jobCategoryID"].(float64))
	description := jobData["description"].(string)
	
	job := models.Job{
		CustomerID:    customerID,
		StatusID:      statusID,
		JobCategoryID: &jobCategoryID,
		Description:   &description,
	}

	if startDate, ok := jobData["startDate"].(string); ok {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			job.StartDate = &parsed
		}
	}

	if endDate, ok := jobData["endDate"].(string); ok {
		if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
			job.EndDate = &parsed
		}
	}

	return h.db.Create(&job).Error
}

// processAssignDevice handles offline device assignment
func (h *PWAHandler) processAssignDevice(action models.OfflineSyncQueue) error {
	var assignData map[string]interface{}
	if err := json.Unmarshal(action.EntityData, &assignData); err != nil {
		return err
	}

	jobID := uint(assignData["jobID"].(float64))
	deviceID := assignData["deviceID"].(string)

	// Check if assignment already exists
	var existing models.JobDevice
	result := h.db.Where("jobID = ? AND deviceID = ?", jobID, deviceID).First(&existing)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Create new assignment
		assignment := models.JobDevice{
			JobID:    jobID,
			DeviceID: deviceID,
		}
		return h.db.Create(&assignment).Error
	}

	// Assignment already exists
	return nil
}

// processUpdateStatus handles offline status updates
func (h *PWAHandler) processUpdateStatus(action models.OfflineSyncQueue) error {
	var statusData map[string]interface{}
	if err := json.Unmarshal(action.EntityData, &statusData); err != nil {
		return err
	}

	entityType := statusData["entityType"].(string)
	entityID := statusData["entityID"].(string)
	newStatus := statusData["status"].(string)

	switch entityType {
	case "device":
		return h.db.Model(&models.Device{}).
			Where("deviceID = ?", entityID).
			Update("status", newStatus).Error
	
	case "case":
		return h.db.Model(&models.Case{}).
			Where("caseID = ?", entityID).
			Update("status", newStatus).Error
	
	default:
		return gorm.ErrRecordNotFound
	}
}

// GetOfflineManifest provides offline capabilities information
func (h *PWAHandler) GetOfflineManifest(c *gin.Context) {
	manifest := map[string]interface{}{
		"version": "2.0.0",
		"offline_capabilities": map[string]interface{}{
			"create_jobs":    true,
			"assign_devices": true,
			"update_status":  true,
			"view_analytics": true,
			"search_cached":  true,
		},
		"cache_duration": map[string]interface{}{
			"static_files": "7d",
			"api_data":     "1h",
			"analytics":    "30m",
		},
		"sync_endpoints": map[string]string{
			"sync_offline": "/pwa/sync",
			"subscribe":    "/pwa/subscribe",
			"unsubscribe":  "/pwa/unsubscribe",
		},
	}

	c.JSON(http.StatusOK, manifest)
}

// InstallPrompt provides PWA installation information
func (h *PWAHandler) InstallPrompt(c *gin.Context) {
	prompt := map[string]interface{}{
		"title":       "Install TS Equipment Manager",
		"description": "Get instant access to your equipment management system",
		"features": []string{
			"Work offline with cached data",
			"Receive push notifications for important updates",
			"Fast loading and native app experience",
			"Home screen access",
			"Background data synchronization",
		},
		"install_steps": []string{
			"Tap the share button in your browser",
			"Select 'Add to Home Screen'",
			"Confirm the installation",
		},
	}

	c.JSON(http.StatusOK, prompt)
}

// GetConnectionStatus provides network status
func (h *PWAHandler) GetConnectionStatus(c *gin.Context) {
	// This endpoint helps the frontend determine connection quality
	c.JSON(http.StatusOK, gin.H{
		"status":    "online",
		"timestamp": time.Now().Unix(),
		"server":    "ts-equipment-manager",
	})
}