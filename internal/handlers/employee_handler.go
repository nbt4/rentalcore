package handlers

import (
	"net/http"
	"strconv"

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
	models.Employee
	SkillIDs []uint `json:"skill_ids"`
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
	if err := h.repo.Create(&req.Employee, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req.Employee)
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
	req.Employee.ID = uint(id)
	if err := h.repo.Update(&req.Employee, req.SkillIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req.Employee)
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
