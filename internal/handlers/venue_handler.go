package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

type VenueHandler struct {
	repo *repository.VenueRepository
}

func NewVenueHandler(repo *repository.VenueRepository) *VenueHandler {
	return &VenueHandler{repo: repo}
}

type venueRequest struct {
	Name        string  `json:"name"`
	Street      *string `json:"street"`
	HouseNumber *string `json:"house_number"`
	ZIP         *string `json:"zip"`
	City        *string `json:"city"`
	ContactName *string `json:"contact_name"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Notes       *string `json:"notes"`
}

func nilIfEmptyStr(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
}

func (h *VenueHandler) List(c *gin.Context) {
	venues, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, venues)
}

func (h *VenueHandler) Create(c *gin.Context) {
	var req venueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	v := &models.Venue{
		Name:        strings.TrimSpace(req.Name),
		Street:      nilIfEmptyStr(req.Street),
		HouseNumber: nilIfEmptyStr(req.HouseNumber),
		ZIP:         nilIfEmptyStr(req.ZIP),
		City:        nilIfEmptyStr(req.City),
		ContactName: nilIfEmptyStr(req.ContactName),
		Phone:       nilIfEmptyStr(req.Phone),
		Email:       nilIfEmptyStr(req.Email),
		Notes:       nilIfEmptyStr(req.Notes),
	}
	if err := h.repo.Create(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *VenueHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req venueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	v := &models.Venue{
		ID:          uint(id),
		Name:        strings.TrimSpace(req.Name),
		Street:      nilIfEmptyStr(req.Street),
		HouseNumber: nilIfEmptyStr(req.HouseNumber),
		ZIP:         nilIfEmptyStr(req.ZIP),
		City:        nilIfEmptyStr(req.City),
		ContactName: nilIfEmptyStr(req.ContactName),
		Phone:       nilIfEmptyStr(req.Phone),
		Email:       nilIfEmptyStr(req.Email),
		Notes:       nilIfEmptyStr(req.Notes),
	}
	if err := h.repo.Update(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VenueHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
