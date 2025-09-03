package handlers

import (
	"net/http"

	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
	statusRepo *repository.StatusRepository
}

func NewStatusHandler(statusRepo *repository.StatusRepository) *StatusHandler {
	return &StatusHandler{statusRepo: statusRepo}
}

func (h *StatusHandler) ListStatuses(c *gin.Context) {
	statuses, err := h.statusRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statuses": statuses})
}

