package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type ScanBoardHandler struct {
	jobRepo    *repository.JobRepository
	deviceRepo *repository.DeviceRepository
	db         *repository.Database
}

func NewScanBoardHandler(jobRepo *repository.JobRepository, deviceRepo *repository.DeviceRepository, db *repository.Database) *ScanBoardHandler {
	return &ScanBoardHandler{
		jobRepo:    jobRepo,
		deviceRepo: deviceRepo,
		db:         db,
	}
}

// GetScanBoardData returns the devices for a specific job for the scan board
func (h *ScanBoardHandler) GetScanBoardData(c *gin.Context) {
	jobIDStr := c.Param("jobID")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get job to verify it exists
	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Get devices for this job with pack status
	devices, err := h.getScanBoardDevices(uint(jobID))
	if err != nil {
		fmt.Printf("Error getting scan board devices: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobID":       job.JobID,
		"description": job.Description,
		"devices":     devices,
	})
}

// ScanDevice handles scanning a device for the pack workflow
func (h *ScanBoardHandler) ScanDevice(c *gin.Context) {
	jobIDStr := c.Param("jobID")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var scanReq models.ScanRequest
	if err := c.ShouldBindJSON(&scanReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Determine device ID from request
	var deviceID string
	if scanReq.DeviceID != nil && *scanReq.DeviceID != "" {
		deviceID = *scanReq.DeviceID
	} else if scanReq.BarcodePayload != nil && *scanReq.BarcodePayload != "" {
		// For now, assume barcode payload is the device ID
		// In a real implementation, you might need to decode the barcode
		deviceID = *scanReq.BarcodePayload
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID or barcode payload required"})
		return
	}

	// Validate that device belongs to this job
	exists, err := h.deviceBelongsToJob(deviceID, uint(jobID))
	if err != nil {
		fmt.Printf("Error checking device job membership: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device not assigned to this job"})
		return
	}

	// Update pack status to 'packed'
	err = h.updatePackStatus(uint(jobID), deviceID, "packed")
	if err != nil {
		fmt.Printf("Error updating pack status: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pack status"})
		return
	}

	// Log the event
	err = h.logDeviceEvent(uint(jobID), deviceID, "scanned", "system")
	if err != nil {
		fmt.Printf("Error logging device event: %v\n", err)
		// Don't fail the request for logging errors
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device scanned successfully",
		"deviceID": deviceID,
	})
}

// FinishPack handles finishing the pack process
func (h *ScanBoardHandler) FinishPack(c *gin.Context) {
	jobIDStr := c.Param("jobID")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var finishReq models.FinishPackRequest
	if err := c.ShouldBindJSON(&finishReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check for missing items
	missingItems, err := h.getMissingItems(uint(jobID))
	if err != nil {
		fmt.Printf("Error getting missing items: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check missing items"})
		return
	}

	// If there are missing items and not forcing, return them
	if len(missingItems) > 0 && !finishReq.Force {
		c.JSON(http.StatusOK, models.FinishPackResponse{
			Success:      false,
			MissingItems: missingItems,
			Message:      "Some items are not yet packed",
		})
		return
	}

	// Mark all remaining items as packed if forcing
	if finishReq.Force && len(missingItems) > 0 {
		err = h.markAllAsPacked(uint(jobID))
		if err != nil {
			fmt.Printf("Error marking all as packed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finish packing"})
			return
		}
	}

	// Log completion event
	err = h.logJobEvent(uint(jobID), "pack_completed")
	if err != nil {
		fmt.Printf("Error logging job completion: %v\n", err)
	}

	c.JSON(http.StatusOK, models.FinishPackResponse{
		Success: true,
		Message: "Pack process completed successfully",
	})
}

// Helper methods

func (h *ScanBoardHandler) getScanBoardDevices(jobID uint) ([]models.ScanBoardDevice, error) {
	query := `
		SELECT
			jd.deviceID,
			COALESCE(p.name, 'Unknown Product') as productName,
			jd.pack_status as packStatus,
			jd.deviceID as barcodePayload,
			pi.file_path as imageUrl
		FROM jobdevices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN product_images pi ON p.productID = pi.productID AND pi.is_primary = 1
		WHERE jd.jobID = ?
		ORDER BY p.name, jd.deviceID
	`

	rows, err := h.db.Raw(query, jobID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.ScanBoardDevice
	for rows.Next() {
		var device models.ScanBoardDevice
		var imageUrl *string
		err := rows.Scan(&device.DeviceID, &device.ProductName, &device.PackStatus, &device.BarcodePayload, &imageUrl)
		if err != nil {
			return nil, err
		}
		if imageUrl != nil && *imageUrl != "" {
			device.ImageURL = imageUrl
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func (h *ScanBoardHandler) deviceBelongsToJob(deviceID string, jobID uint) (bool, error) {
	var count int64
	err := h.db.Table("jobdevices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Count(&count).Error

	return count > 0, err
}

func (h *ScanBoardHandler) updatePackStatus(jobID uint, deviceID string, status string) error {
	now := time.Now()
	return h.db.Table("jobdevices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Updates(map[string]interface{}{
			"pack_status": status,
			"pack_ts":     now,
		}).Error
}

func (h *ScanBoardHandler) logDeviceEvent(jobID uint, deviceID string, eventType string, actor string) error {
	event := models.JobDeviceEvent{
		JobID:     jobID,
		DeviceID:  deviceID,
		EventType: eventType,
		Actor:     &actor,
		Timestamp: time.Now(),
	}

	return h.db.Create(&event).Error
}

func (h *ScanBoardHandler) logJobEvent(jobID uint, eventType string) error {
	event := models.JobDeviceEvent{
		JobID:     jobID,
		DeviceID:  "", // Empty for job-level events
		EventType: eventType,
		Actor:     nil,
		Timestamp: time.Now(),
	}

	return h.db.Create(&event).Error
}

func (h *ScanBoardHandler) getMissingItems(jobID uint) ([]string, error) {
	query := `
		SELECT
			CONCAT(COALESCE(p.name, 'Unknown Product'), ' (', jd.deviceID, ')') as missing_item
		FROM jobdevices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		WHERE jd.jobID = ? AND jd.pack_status = 'pending'
		ORDER BY p.name, jd.deviceID
	`

	rows, err := h.db.Raw(query, jobID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var missing []string
	for rows.Next() {
		var item string
		err := rows.Scan(&item)
		if err != nil {
			return nil, err
		}
		missing = append(missing, item)
	}

	return missing, nil
}

func (h *ScanBoardHandler) markAllAsPacked(jobID uint) error {
	now := time.Now()
	return h.db.Table("jobdevices").
		Where("jobID = ? AND pack_status = 'pending'", jobID).
		Updates(map[string]interface{}{
			"pack_status": "packed",
			"pack_ts":     now,
		}).Error
}