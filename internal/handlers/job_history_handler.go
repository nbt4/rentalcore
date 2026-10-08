package handlers

import (
	"net/http"
	"strconv"

	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// JobHistoryHandler handles job history related endpoints
type JobHistoryHandler struct {
	DB             *gorm.DB
	HistoryService *services.JobHistoryService
}

// NewJobHistoryHandler creates a new job history handler
func NewJobHistoryHandler(db *gorm.DB) *JobHistoryHandler {
	return &JobHistoryHandler{
		DB:             db,
		HistoryService: services.NewJobHistoryService(db),
	}
}

// GetJobHistory returns the history for a specific job
// GET /api/jobs/:id/history
func (h *JobHistoryHandler) GetJobHistory(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	history, err := h.HistoryService.GetJobHistory(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"history": history,
	})
}
