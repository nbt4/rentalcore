package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
)

type EmployeeHandler struct {
	repo *repository.EmployeeRepository
}

func NewEmployeeHandler(repo *repository.EmployeeRepository) *EmployeeHandler {
	return &EmployeeHandler{repo: repo}
}

type employeeRequest struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	Mobile      *string `json:"mobile"`
	Street      *string `json:"street"`
	HouseNumber *string `json:"house_number"`
	ZIP         *string `json:"zip"`
	City        *string `json:"city"`
	Country     *string `json:"country"`
	DateOfBirth *string `json:"date_of_birth"`
	IBAN        *string `json:"iban"`
	Notes       *string `json:"notes"`
	IsActive    bool    `json:"is_active"`
	SkillIDs    []uint  `json:"skill_ids"`
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func (r *employeeRequest) toEmployee() (models.Employee, error) {
	e := models.Employee{
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		Email:       nilIfEmpty(r.Email),
		Phone:       nilIfEmpty(r.Phone),
		Mobile:      nilIfEmpty(r.Mobile),
		Street:      nilIfEmpty(r.Street),
		HouseNumber: nilIfEmpty(r.HouseNumber),
		ZIP:         nilIfEmpty(r.ZIP),
		City:        nilIfEmpty(r.City),
		Country:     nilIfEmpty(r.Country),
		IBAN:        nilIfEmpty(r.IBAN),
		Notes:       nilIfEmpty(r.Notes),
		IsActive:    r.IsActive,
	}
	if r.DateOfBirth != nil && *r.DateOfBirth != "" {
		t, err := time.Parse("2006-01-02", *r.DateOfBirth)
		if err != nil {
			return e, fmt.Errorf("invalid date_of_birth: %w", err)
		}
		e.DateOfBirth = &t
	}
	return e, nil
}

func (h *EmployeeHandler) List(c *gin.Context) {
	employees, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, employees)
}

func (h *EmployeeHandler) ListActive(c *gin.Context) {
	employees, err := h.repo.ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, employees)
}

func (h *EmployeeHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	e, err := h.repo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *EmployeeHandler) Create(c *gin.Context) {
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	emp, err := req.toEmployee()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Create(&emp, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, emp)
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	emp, err := req.toEmployee()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	emp.ID = uint(id)
	if err := h.repo.Update(&emp, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, emp)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
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
