package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go-barcode-webapp/internal/models"
)

type M365SettingsHandler struct {
	db *gorm.DB
}

func NewM365SettingsHandler(db *gorm.DB) *M365SettingsHandler {
	return &M365SettingsHandler{db: db}
}

func (h *M365SettingsHandler) GetM365Settings(c *gin.Context) {
	var s models.M365Settings
	if err := h.db.First(&s).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tenantId":        os.Getenv("M365_TENANT_ID"),
			"clientId":        os.Getenv("M365_CLIENT_ID"),
			"clientSecret":    "",
			"mailboxId":       os.Getenv("M365_SHARED_MAILBOX_ID"),
			"syncInterval":    firstNonEmpty(os.Getenv("M365_SYNC_INTERVAL"), "5m"),
			"calendarMailbox": firstNonEmpty(os.Getenv("M365_CALENDAR_MAILBOX"), "events@tsunami-events.de"),
			"source":          "env",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tenantId":        s.TenantID,
		"clientId":        s.ClientID,
		"clientSecret":    maskSecret(s.ClientSecret),
		"mailboxId":       s.MailboxID,
		"syncInterval":    s.SyncInterval,
		"calendarMailbox": s.CalendarMailbox,
		"source":          "db",
		"updatedAt":       s.UpdatedAt,
	})
}

func (h *M365SettingsHandler) UpdateM365Settings(c *gin.Context) {
	var req struct {
		TenantID        string `json:"tenantId"`
		ClientID        string `json:"clientId"`
		ClientSecret    string `json:"clientSecret"`
		MailboxID       string `json:"mailboxId"`
		SyncInterval    string `json:"syncInterval"`
		CalendarMailbox string `json:"calendarMailbox"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ungültige Anfrage"})
		return
	}

	var s models.M365Settings
	h.db.First(&s)

	s.TenantID = req.TenantID
	s.ClientID = req.ClientID
	if req.ClientSecret != "" && req.ClientSecret != "••••••••" {
		s.ClientSecret = req.ClientSecret
	}
	s.MailboxID = req.MailboxID
	if req.SyncInterval != "" {
		s.SyncInterval = req.SyncInterval
	}
	s.CalendarMailbox = req.CalendarMailbox

	if s.ID == 0 {
		if err := h.db.Create(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := h.db.Save(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Gespeichert"})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "••••••••"
}
