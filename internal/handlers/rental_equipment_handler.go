package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type RentalEquipmentHandler struct {
	repo *repository.RentalEquipmentRepository
}

func NewRentalEquipmentHandler(repo *repository.RentalEquipmentRepository) *RentalEquipmentHandler {
	return &RentalEquipmentHandler{repo: repo}
}

// ShowRentalEquipmentList displays the rental equipment management page
func (h *RentalEquipmentHandler) ShowRentalEquipmentList(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	var rentalEquipment []models.RentalEquipment
	err := h.repo.GetAllRentalEquipment(&rentalEquipment)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_page.html", gin.H{
			"error_code":    500,
			"error_message": "Database Error",
			"error_details": fmt.Sprintf("Failed to load rental equipment: %v", err),
			"user":          user,
		})
		return
	}

	c.HTML(http.StatusOK, "rental_equipment_standalone.html", gin.H{
		"title":           "Rental Equipment Management",
		"user":            user,
		"rentalEquipment": rentalEquipment,
		"currentPage":     "rental-equipment",
	})
}

// ShowRentalEquipmentForm displays the form for creating/editing rental equipment
func (h *RentalEquipmentHandler) ShowRentalEquipmentForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	equipmentID := c.Param("id")

	var rentalEquipment *models.RentalEquipment
	var isEdit bool

	if equipmentID != "" {
		isEdit = true
		id, err := strconv.ParseUint(equipmentID, 10, 32)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error_page.html", gin.H{
				"error_code":    400,
				"error_message": "Invalid Equipment ID",
				"error_details": "The equipment ID provided is not valid",
				"user":          user,
			})
			return
		}

		var equipment models.RentalEquipment
		err = h.repo.GetRentalEquipmentByID(uint(id), &equipment)
		if err != nil {
			c.HTML(http.StatusNotFound, "error_page.html", gin.H{
				"error_code":    404,
				"error_message": "Equipment Not Found",
				"error_details": "The requested rental equipment could not be found",
				"user":          user,
			})
			return
		}
		rentalEquipment = &equipment
	}

	c.HTML(http.StatusOK, "rental_equipment_form_standalone.html", gin.H{
		"title":           "Rental Equipment Form",
		"user":            user,
		"rentalEquipment": rentalEquipment,
		"isEdit":          isEdit,
		"currentPage":     "rental-equipment",
	})
}

// CreateRentalEquipment creates a new rental equipment item
func (h *RentalEquipmentHandler) CreateRentalEquipment(c *gin.Context) {
	var request models.CreateRentalEquipmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, _ := GetCurrentUser(c)
	var createdBy *uint
	if user != nil {
		createdBy = &user.UserID
	}

	rentalEquipment := &models.RentalEquipment{
		ProductName:  request.ProductName,
		SupplierName: request.SupplierName,
		RentalPrice:  request.RentalPrice,
		Category:     request.Category,
		Description:  request.Description,
		Notes:        request.Notes,
		IsActive:     request.IsActive,
		CreatedBy:    createdBy,
	}

	err := h.repo.CreateRentalEquipment(rentalEquipment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create rental equipment: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Rental equipment created successfully",
		"rentalEquipment": rentalEquipment,
	})
}

// UpdateRentalEquipment updates an existing rental equipment item
func (h *RentalEquipmentHandler) UpdateRentalEquipment(c *gin.Context) {
	equipmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment ID"})
		return
	}

	var request models.UpdateRentalEquipmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing equipment
	var rentalEquipment models.RentalEquipment
	err = h.repo.GetRentalEquipmentByID(uint(equipmentID), &rentalEquipment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rental equipment not found"})
		return
	}

	// Update fields
	rentalEquipment.ProductName = request.ProductName
	rentalEquipment.SupplierName = request.SupplierName
	rentalEquipment.RentalPrice = request.RentalPrice
	rentalEquipment.Category = request.Category
	rentalEquipment.Description = request.Description
	rentalEquipment.Notes = request.Notes
	rentalEquipment.IsActive = request.IsActive

	err = h.repo.UpdateRentalEquipment(&rentalEquipment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update rental equipment: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Rental equipment updated successfully",
		"rentalEquipment": rentalEquipment,
	})
}

// DeleteRentalEquipment deletes a rental equipment item
func (h *RentalEquipmentHandler) DeleteRentalEquipment(c *gin.Context) {
	equipmentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment ID"})
		return
	}

	err = h.repo.DeleteRentalEquipment(uint(equipmentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete rental equipment: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rental equipment deleted successfully"})
}

// GetRentalEquipmentAPI returns rental equipment data via API
func (h *RentalEquipmentHandler) GetRentalEquipmentAPI(c *gin.Context) {
	var rentalEquipment []models.RentalEquipment
	err := h.repo.GetAllRentalEquipment(&rentalEquipment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get rental equipment: %v", err)})
		return
	}

	// Convert to response format
	var responses []models.RentalEquipmentResponse
	for _, equipment := range rentalEquipment {
		responses = append(responses, models.RentalEquipmentResponse{
			EquipmentID:   equipment.EquipmentID,
			ProductName:   equipment.ProductName,
			SupplierName:  equipment.SupplierName,
			RentalPrice:   equipment.RentalPrice,
			Category:      equipment.Category,
			Description:   equipment.Description,
			Notes:         equipment.Notes,
			IsActive:      equipment.IsActive,
			CreatedAt:     equipment.CreatedAt,
			UpdatedAt:     equipment.UpdatedAt,
			Creator:       equipment.Creator,
			TotalUsed:     equipment.TotalUsed,
			TotalRevenue:  equipment.TotalRevenue,
			LastUsedDate:  equipment.LastUsedDate,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// AddRentalToJob adds existing rental equipment to a job
func (h *RentalEquipmentHandler) AddRentalToJob(c *gin.Context) {
	var request models.AddRentalToJobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get rental price from equipment
	var equipment models.RentalEquipment
	err := h.repo.GetRentalEquipmentByID(request.EquipmentID, &equipment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rental equipment not found"})
		return
	}

	// Calculate total cost
	totalCost := equipment.RentalPrice * float64(request.Quantity) * float64(request.DaysUsed)

	jobRental := &models.JobRentalEquipment{
		JobID:       request.JobID,
		EquipmentID: request.EquipmentID,
		Quantity:    request.Quantity,
		DaysUsed:    request.DaysUsed,
		TotalCost:   totalCost,
		Notes:       request.Notes,
	}

	err = h.repo.AddRentalToJob(jobRental)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add rental to job: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Rental equipment added to job successfully",
		"jobRental":  jobRental,
		"totalCost":  totalCost,
	})
}

// CreateManualRentalEntry creates new rental equipment and adds it to a job in one step
func (h *RentalEquipmentHandler) CreateManualRentalEntry(c *gin.Context) {
	var request models.ManualRentalEntryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, _ := GetCurrentUser(c)
	var createdBy *uint
	if user != nil {
		createdBy = &user.UserID
	}

	equipment, jobRental, err := h.repo.CreateRentalEquipmentFromManualEntry(&request, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create manual rental entry: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Manual rental entry created successfully",
		"rentalEquipment":   equipment,
		"jobRental":         jobRental,
		"totalCost":         jobRental.TotalCost,
	})
}

// GetJobRentalEquipment returns rental equipment for a specific job
func (h *RentalEquipmentHandler) GetJobRentalEquipment(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var jobRentals []models.JobRentalEquipment
	err = h.repo.GetJobRentalEquipment(uint(jobID), &jobRentals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get job rental equipment: %v", err)})
		return
	}

	c.JSON(http.StatusOK, jobRentals)
}

// RemoveRentalFromJob removes rental equipment from a job
func (h *RentalEquipmentHandler) RemoveRentalFromJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	equipmentID, err := strconv.ParseUint(c.Param("equipmentId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment ID"})
		return
	}

	err = h.repo.RemoveRentalFromJob(uint(jobID), uint(equipmentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to remove rental from job: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rental equipment removed from job successfully"})
}

// ShowRentalAnalytics displays the rental equipment analytics page
func (h *RentalEquipmentHandler) ShowRentalAnalytics(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	analytics, err := h.repo.GetRentalEquipmentAnalytics()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_page.html", gin.H{
			"error_code":    500,
			"error_message": "Analytics Error",
			"error_details": fmt.Sprintf("Failed to load rental analytics: %v", err),
			"user":          user,
		})
		return
	}

	c.HTML(http.StatusOK, "rental_equipment_analytics_standalone.html", gin.H{
		"title":       "Rental Equipment Analytics",
		"user":        user,
		"analytics":   analytics,
		"currentPage": "rental-analytics",
	})
}

// GetRentalAnalyticsAPI returns rental analytics data via API
func (h *RentalEquipmentHandler) GetRentalAnalyticsAPI(c *gin.Context) {
	analytics, err := h.repo.GetRentalEquipmentAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get rental analytics: %v", err)})
		return
	}

	c.JSON(http.StatusOK, analytics)
}