package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"

	"go-barcode-webapp/internal/logger"
)

type AccessoriesConsumablesHandler struct {
	repo *repository.AccessoriesConsumablesRepository
}

func NewAccessoriesConsumablesHandler(repo *repository.AccessoriesConsumablesRepository) *AccessoriesConsumablesHandler {
	return &AccessoriesConsumablesHandler{repo: repo}
}

// ============================================================================

// Count Types API
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetCountTypesAPI(c *gin.Context) {
	countTypes, err := h.repo.GetAllCountTypes()
	if err != nil {
		logger.LogInfo("❌ Error fetching count types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch count types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count_types": countTypes})
}

// ============================================================================
// Product Dependencies API (WarehouseCore integration)
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetProductDependenciesAPI(c *gin.Context) {
	productIDStr := c.Param("productid")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	dependencies, err := h.repo.GetProductDependencies(uint(productID))
	if err != nil {
		logger.LogInfo("❌ Error fetching product dependencies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product dependencies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dependencies": dependencies})
}

// ============================================================================
// Product Accessories API
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetProductAccessoriesAPI(c *gin.Context) {
	productIDStr := c.Param("productid")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	accessories, err := h.repo.GetProductAccessories(uint(productID))
	if err != nil {
		logger.LogInfo("❌ Error fetching product accessories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product accessories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessories": accessories})
}

func (h *AccessoriesConsumablesHandler) AddProductAccessoryAPI(c *gin.Context) {
	var pa models.ProductAccessory
	if err := c.ShouldBindJSON(&pa); err != nil {
		logger.LogInfo("❌ Error binding product accessory JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	if err := h.repo.AddProductAccessory(&pa); err != nil {
		logger.LogInfo("❌ Error adding product accessory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add accessory"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Accessory added successfully"})
}

func (h *AccessoriesConsumablesHandler) RemoveProductAccessoryAPI(c *gin.Context) {
	productIDStr := c.Param("productid")
	accessoryIDStr := c.Param("accessoryID")

	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	accessoryID, err := strconv.ParseUint(accessoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid accessory ID"})
		return
	}

	if err := h.repo.RemoveProductAccessory(uint(productID), uint(accessoryID)); err != nil {
		logger.LogInfo("❌ Error removing product accessory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove accessory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Accessory removed successfully"})
}

func (h *AccessoriesConsumablesHandler) GetAccessoryProductsAPI(c *gin.Context) {
	products, err := h.repo.GetAccessoryProducts()
	if err != nil {
		logger.LogInfo("❌ Error fetching accessory products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch accessory products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

// ============================================================================
// Product Consumables API
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetProductConsumablesAPI(c *gin.Context) {
	productIDStr := c.Param("productid")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	consumables, err := h.repo.GetProductConsumables(uint(productID))
	if err != nil {
		logger.LogInfo("❌ Error fetching product consumables: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product consumables"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"consumables": consumables})
}

func (h *AccessoriesConsumablesHandler) AddProductConsumableAPI(c *gin.Context) {
	var pc models.ProductConsumable
	if err := c.ShouldBindJSON(&pc); err != nil {
		logger.LogInfo("❌ Error binding product consumable JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	if err := h.repo.AddProductConsumable(&pc); err != nil {
		logger.LogInfo("❌ Error adding product consumable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add consumable"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Consumable added successfully"})
}

func (h *AccessoriesConsumablesHandler) RemoveProductConsumableAPI(c *gin.Context) {
	productIDStr := c.Param("productid")
	consumableIDStr := c.Param("consumableID")

	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	consumableID, err := strconv.ParseUint(consumableIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consumable ID"})
		return
	}

	if err := h.repo.RemoveProductConsumable(uint(productID), uint(consumableID)); err != nil {
		logger.LogInfo("❌ Error removing product consumable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove consumable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consumable removed successfully"})
}

func (h *AccessoriesConsumablesHandler) GetConsumableProductsAPI(c *gin.Context) {
	products, err := h.repo.GetConsumableProducts()
	if err != nil {
		logger.LogInfo("❌ Error fetching consumable products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch consumable products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

// ============================================================================
// Job Accessories API
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetJobAccessoriesAPI(c *gin.Context) {
	jobIDStr := c.Param("jobid")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	accessories, err := h.repo.GetJobAccessories(uint(jobID))
	if err != nil {
		logger.LogInfo("❌ Error fetching job accessories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job accessories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessories": accessories})
}

func (h *AccessoriesConsumablesHandler) CreateJobAccessoryAPI(c *gin.Context) {
	var ja models.JobAccessory
	if err := c.ShouldBindJSON(&ja); err != nil {
		logger.LogInfo("❌ Error binding job accessory JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	if err := h.repo.CreateJobAccessory(&ja); err != nil {
		logger.LogInfo("❌ Error creating job accessory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job accessory"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job_accessory": ja})
}

func (h *AccessoriesConsumablesHandler) UpdateJobAccessoryAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid accessory ID"})
		return
	}

	var ja models.JobAccessory
	if err := c.ShouldBindJSON(&ja); err != nil {
		logger.LogInfo("❌ Error binding job accessory JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	ja.JobAccessoryID = id
	if err := h.repo.UpdateJobAccessory(&ja); err != nil {
		logger.LogInfo("❌ Error updating job accessory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job accessory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job_accessory": ja})
}

func (h *AccessoriesConsumablesHandler) DeleteJobAccessoryAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid accessory ID"})
		return
	}

	if err := h.repo.DeleteJobAccessory(id); err != nil {
		logger.LogInfo("❌ Error deleting job accessory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job accessory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job accessory deleted successfully"})
}

// ============================================================================
// Job Consumables API
// ============================================================================

func (h *AccessoriesConsumablesHandler) GetJobConsumablesAPI(c *gin.Context) {
	jobIDStr := c.Param("jobid")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	consumables, err := h.repo.GetJobConsumables(uint(jobID))
	if err != nil {
		logger.LogInfo("❌ Error fetching job consumables: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job consumables"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"consumables": consumables})
}

func (h *AccessoriesConsumablesHandler) CreateJobConsumableAPI(c *gin.Context) {
	var jc models.JobConsumable
	if err := c.ShouldBindJSON(&jc); err != nil {
		logger.LogInfo("❌ Error binding job consumable JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	if err := h.repo.CreateJobConsumable(&jc); err != nil {
		logger.LogInfo("❌ Error creating job consumable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job consumable"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job_consumable": jc})
}

func (h *AccessoriesConsumablesHandler) UpdateJobConsumableAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consumable ID"})
		return
	}

	var jc models.JobConsumable
	if err := c.ShouldBindJSON(&jc); err != nil {
		logger.LogInfo("❌ Error binding job consumable JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	jc.JobConsumableID = id
	if err := h.repo.UpdateJobConsumable(&jc); err != nil {
		logger.LogInfo("❌ Error updating job consumable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job consumable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job_consumable": jc})
}

func (h *AccessoriesConsumablesHandler) DeleteJobConsumableAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consumable ID"})
		return
	}

	if err := h.repo.DeleteJobConsumable(id); err != nil {
		logger.LogInfo("❌ Error deleting job consumable: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job consumable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job consumable deleted successfully"})
}

// ============================================================================

// Scanning API (for WarehouseCore integration)
// ============================================================================

func (h *AccessoriesConsumablesHandler) ScanAccessoryAPI(c *gin.Context) {
	var req struct {
		Barcode   string `json:"barcode" binding:"required"`
		JobID     uint   `json:"job_id" binding:"required"`
		Direction string `json:"direction" binding:"required"` // "out" or "in"
		Quantity  int    `json:"quantity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	if req.Quantity <= 0 {
		req.Quantity = 1 // Default to 1 if not specified
	}

	// Find accessory by barcode
	product, err := h.repo.GetAccessoryByBarcode(req.Barcode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Accessory not found"})
		return
	}

	// Find job accessories for this job and product
	jobAccessories, err := h.repo.GetJobAccessoriesByJobAndProduct(req.JobID, product.ProductID)
	if err != nil || len(jobAccessories) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No accessories assigned for this job"})
		return
	}

	// Use the first matching job accessory
	ja := jobAccessories[0]

	// Get current user ID
	var userID *uint64
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			uid := uint64(u.UserID)
			userID = &uid
		}
	}

	// Perform scan
	if req.Direction == "out" {
		if err := h.repo.ScanAccessoryOut(ja.JobAccessoryID, req.Quantity, userID); err != nil {
			logger.LogInfo("❌ Error scanning accessory out: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.Direction == "in" {
		if err := h.repo.ScanAccessoryIn(ja.JobAccessoryID, req.Quantity, userID); err != nil {
			logger.LogInfo("❌ Error scanning accessory in: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid direction (must be 'out' or 'in')"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Accessory scanned successfully",
		"product":         product,
		"quantity":        req.Quantity,
		"remaining_stock": product.StockQuantity,
	})
}

func (h *AccessoriesConsumablesHandler) ScanConsumableAPI(c *gin.Context) {
	var req struct {
		Barcode   string  `json:"barcode" binding:"required"`
		JobID     uint    `json:"job_id" binding:"required"`
		Direction string  `json:"direction" binding:"required"` // "out" or "in"
		Quantity  float64 `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid data: %v", err)})
		return
	}

	// Find consumable by barcode
	product, err := h.repo.GetConsumableByBarcode(req.Barcode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consumable not found"})
		return
	}

	// Find job consumables for this job and product
	jobConsumables, err := h.repo.GetJobConsumablesByJobAndProduct(req.JobID, product.ProductID)
	if err != nil || len(jobConsumables) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No consumables assigned for this job"})
		return
	}

	// Use the first matching job consumable
	jc := jobConsumables[0]

	// Get current user ID
	var userID *uint64
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			uid := uint64(u.UserID)
			userID = &uid
		}
	}

	// Perform scan
	if req.Direction == "out" {
		if err := h.repo.ScanConsumableOut(jc.JobConsumableID, req.Quantity, userID); err != nil {
			logger.LogInfo("❌ Error scanning consumable out: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.Direction == "in" {
		if err := h.repo.ScanConsumableIn(jc.JobConsumableID, req.Quantity, userID); err != nil {
			logger.LogInfo("❌ Error scanning consumable in: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid direction (must be 'out' or 'in')"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Consumable scanned successfully",
		"product":         product,
		"quantity":        req.Quantity,
		"remaining_stock": product.StockQuantity,
	})
}
