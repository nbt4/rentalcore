package handlers

import (
	"crypto/md5"
	"fmt"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type JobAttachmentHandler struct {
	repo       *repository.JobAttachmentRepository
	jobRepo    *repository.JobRepository
	uploadPath string
	maxFileSize int64
}

func NewJobAttachmentHandler(repo *repository.JobAttachmentRepository, jobRepo *repository.JobRepository) *JobAttachmentHandler {
	// Default upload path and max file size (50MB)
	uploadPath := "./uploads/job_attachments"
	maxFileSize := int64(50 << 20) // 50MB

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
	}

	return &JobAttachmentHandler{
		repo:        repo,
		jobRepo:     jobRepo,
		uploadPath:  uploadPath,
		maxFileSize: maxFileSize,
	}
}

// UploadAttachment handles file upload for job attachments
func (h *JobAttachmentHandler) UploadAttachment(c *gin.Context) {
	jobIDStr := c.PostForm("jobID")
	if jobIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Verify job exists
	_, err = h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		log.Printf("Job not found for ID %d: %v", jobID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Get uploaded file
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Error getting uploaded file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Check file size
	if fileHeader.Size > h.maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File too large. Maximum size is %d MB", h.maxFileSize/(1024*1024)),
		})
		return
	}

	// Get description
	description := c.PostForm("description")

	// Generate unique filename
	originalFilename := fileHeader.Filename
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Format("20060102_150405")
	uniqueFilename := fmt.Sprintf("job_%d_%s_%x%s", jobID, timestamp,
		md5.Sum([]byte(originalFilename+time.Now().String())), ext)

	// Create full file path
	fullPath := filepath.Join(h.uploadPath, uniqueFilename)

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creating destination file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	// Copy file content
	fileSize, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("Error copying file content: %v", err)
		// Clean up created file
		os.Remove(fullPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Detect MIME type
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get current user ID from session
	userID := h.getCurrentUserID(c)

	// Create attachment record
	attachment := &models.JobAttachment{
		JobID:            uint(jobID),
		Filename:         uniqueFilename,
		OriginalFilename: originalFilename,
		FilePath:         fullPath,
		FileSize:         fileSize,
		MimeType:         mimeType,
		UploadedBy:       userID,
		UploadedAt:       time.Now(),
		Description:      description,
		IsActive:         true,
	}

	err = h.repo.Create(attachment)
	if err != nil {
		log.Printf("Error saving attachment to database: %v", err)
		// Clean up created file
		os.Remove(fullPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save attachment"})
		return
	}

	log.Printf("✅ Successfully uploaded attachment %s for job %d", originalFilename, jobID)

	c.JSON(http.StatusCreated, gin.H{
		"message":      "File uploaded successfully",
		"attachmentID": attachment.AttachmentID,
		"filename":     originalFilename,
	})
}

// GetJobAttachments returns all attachments for a job
func (h *JobAttachmentHandler) GetJobAttachments(c *gin.Context) {
	jobIDStr := c.Param("jobid")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	attachments, err := h.repo.GetByJobID(uint(jobID))
	if err != nil {
		log.Printf("Error getting attachments for job %d: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attachments"})
		return
	}

	// Convert to response format
	var responses []models.JobAttachmentResponse
	for _, attachment := range attachments {
		response := models.JobAttachmentResponse{
			AttachmentID:      attachment.AttachmentID,
			JobID:             attachment.JobID,
			Filename:          attachment.Filename,
			OriginalFilename:  attachment.OriginalFilename,
			FileSize:          attachment.FileSize,
			MimeType:          attachment.MimeType,
			UploadedBy:        attachment.UploadedBy,
			UploadedAt:        attachment.UploadedAt,
			Description:       attachment.Description,
			IsActive:          attachment.IsActive,
			Uploader:          attachment.Uploader,
			FileSizeFormatted: h.formatFileSize(attachment.FileSize),
			IsImage:           h.isImageMimeType(attachment.MimeType),
		}
		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, responses)
}

// ViewAttachment serves a file for inline viewing (preview)
func (h *JobAttachmentHandler) ViewAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("id")
	attachmentID, err := strconv.ParseUint(attachmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	attachment, err := h.repo.GetByID(uint(attachmentID))
	if err != nil {
		log.Printf("Attachment not found for ID %d: %v", attachmentID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Check if file exists
	if _, err := os.Stat(attachment.FilePath); os.IsNotExist(err) {
		log.Printf("File not found on disk: %s", attachment.FilePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Set headers for inline viewing (no attachment disposition)
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.FileSize))

	// For PDFs, add additional headers for better browser support
	if attachment.MimeType == "application/pdf" {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", attachment.OriginalFilename))
		c.Header("X-Content-Type-Options", "nosniff")
	} else {
		// For other files that can be displayed inline, set inline disposition
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", attachment.OriginalFilename))
	}

	// Serve the file
	c.File(attachment.FilePath)
}

// DownloadAttachment serves a file download
func (h *JobAttachmentHandler) DownloadAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("id")
	attachmentID, err := strconv.ParseUint(attachmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	attachment, err := h.repo.GetByID(uint(attachmentID))
	if err != nil {
		log.Printf("Attachment not found for ID %d: %v", attachmentID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Check if file exists
	if _, err := os.Stat(attachment.FilePath); os.IsNotExist(err) {
		log.Printf("File not found on disk: %s", attachment.FilePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.OriginalFilename))
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.FileSize))

	// Serve the file
	c.File(attachment.FilePath)
}

// DeleteAttachment soft deletes an attachment
func (h *JobAttachmentHandler) DeleteAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("id")
	attachmentID, err := strconv.ParseUint(attachmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// Get attachment to verify it exists
	attachment, err := h.repo.GetByID(uint(attachmentID))
	if err != nil {
		log.Printf("Attachment not found for ID %d: %v", attachmentID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Soft delete (set is_active to false)
	err = h.repo.Delete(uint(attachmentID))
	if err != nil {
		log.Printf("Error deleting attachment %d: %v", attachmentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attachment"})
		return
	}

	log.Printf("✅ Successfully deleted attachment %s (ID: %d)", attachment.OriginalFilename, attachmentID)

	c.JSON(http.StatusOK, gin.H{"message": "Attachment deleted successfully"})
}

// UpdateAttachmentDescription updates the description of an attachment
func (h *JobAttachmentHandler) UpdateAttachmentDescription(c *gin.Context) {
	attachmentIDStr := c.Param("id")
	attachmentID, err := strconv.ParseUint(attachmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	var req struct {
		Description string `json:"description" binding:"max=1000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attachment, err := h.repo.GetByID(uint(attachmentID))
	if err != nil {
		log.Printf("Attachment not found for ID %d: %v", attachmentID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	attachment.Description = req.Description
	err = h.repo.Update(attachment)
	if err != nil {
		log.Printf("Error updating attachment description for ID %d: %v", attachmentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attachment"})
		return
	}

	log.Printf("✅ Successfully updated description for attachment %d", attachmentID)

	c.JSON(http.StatusOK, gin.H{"message": "Description updated successfully"})
}

// Helper functions

func (h *JobAttachmentHandler) getCurrentUserID(c *gin.Context) *uint {
	// Try to get user ID from session or JWT token
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(uint); ok {
			return &id
		}
	}

	// Try to get from session
	session := getSession(c)
	if session != nil {
		if userID, ok := session["userID"].(uint); ok {
			return &userID
		}
	}

	return nil
}

func (h *JobAttachmentHandler) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (h *JobAttachmentHandler) isImageMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// getSession helper function (needs to be implemented based on your session management)
func getSession(c *gin.Context) map[string]interface{} {
	// This should be implemented based on your session management system
	// For now, return nil - you may need to adjust this based on your setup
	return nil
}