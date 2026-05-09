package handlers

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type PositionHandler struct {
	positionRepo *repository.PositionRepository
	jobRepo      *repository.JobRepository
}

func NewPositionHandler(positionRepo *repository.PositionRepository, jobRepo *repository.JobRepository) *PositionHandler {
	return &PositionHandler{
		positionRepo: positionRepo,
		jobRepo:      jobRepo,
	}
}

func (h *PositionHandler) GetPositions(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

type CreatePositionInput struct {
	PositionType      string   `json:"position_type" binding:"required,oneof=product service rental package"`
	ProductID         *uint    `json:"product_id"`
	ServiceItemID     *uint    `json:"service_item_id"`
	RentalEquipmentID *uint    `json:"rental_equipment_id"`
	Description       string   `json:"description"`
	Quantity          float64  `json:"quantity"`
	Unit              string   `json:"unit"`
	UnitPrice         float64  `json:"unit_price"`
	FollowDayFactor   *float64 `json:"follow_day_factor"`
	DiscountPercent   float64  `json:"discount_percent"`
	DiscountAmount    float64  `json:"discount_amount"`
	TaxRate           *float64 `json:"tax_rate"`
}

func (h *PositionHandler) CreatePosition(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var input CreatePositionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	if input.Unit == "" {
		input.Unit = "Stück"
	}

	followDayFactor := 0.5
	if input.FollowDayFactor != nil {
		followDayFactor = *input.FollowDayFactor
	}
	if input.PositionType == "service" {
		followDayFactor = 0
	}
	// rental and package have no follow_day_factor
	if input.PositionType == "rental" || input.PositionType == "package" {
		followDayFactor = 0
	}

	taxRate := 19.0
	if input.TaxRate != nil {
		taxRate = *input.TaxRate
	}

	nextOrder, _ := h.positionRepo.GetNextSortOrder(uint(jobID))

	pos := models.JobPosition{
		JobID:             uint(jobID),
		PositionType:      input.PositionType,
		ProductID:         input.ProductID,
		ServiceItemID:     input.ServiceItemID,
		RentalEquipmentID: input.RentalEquipmentID,
		Description:       input.Description,
		Quantity:          input.Quantity,
		Unit:              input.Unit,
		UnitPrice:         input.UnitPrice,
		FollowDayFactor:   followDayFactor,
		DiscountPercent:   input.DiscountPercent,
		DiscountAmount:    input.DiscountAmount,
		TaxRate:           taxRate,
		SortOrder:         nextOrder,
	}

	if err := h.positionRepo.Create(&pos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	created, _ := h.positionRepo.GetByID(pos.PositionID)
	c.JSON(http.StatusCreated, gin.H{"position": created})
}

type UpdatePositionInput struct {
	Description     *string  `json:"description"`
	Quantity        *float64 `json:"quantity"`
	Unit            *string  `json:"unit"`
	UnitPrice       *float64 `json:"unit_price"`
	FollowDayFactor *float64 `json:"follow_day_factor"`
	DiscountPercent *float64 `json:"discount_percent"`
	DiscountAmount  *float64 `json:"discount_amount"`
	TaxRate         *float64 `json:"tax_rate"`
}

func (h *PositionHandler) UpdatePosition(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	pos, err := h.positionRepo.GetByID(uint(posID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}

	var input UpdatePositionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Description != nil {
		pos.Description = *input.Description
	}
	if input.Quantity != nil {
		pos.Quantity = *input.Quantity
	}
	if input.Unit != nil {
		pos.Unit = *input.Unit
	}
	if input.UnitPrice != nil {
		pos.UnitPrice = *input.UnitPrice
	}
	if input.FollowDayFactor != nil {
		pos.FollowDayFactor = *input.FollowDayFactor
	}
	if input.DiscountPercent != nil {
		pos.DiscountPercent = *input.DiscountPercent
	}
	if input.DiscountAmount != nil {
		pos.DiscountAmount = *input.DiscountAmount
	}
	if input.TaxRate != nil {
		pos.TaxRate = *input.TaxRate
	}
	pos.UpdatedAt = time.Now()

	if err := h.positionRepo.Update(pos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := h.positionRepo.GetByID(pos.PositionID)
	c.JSON(http.StatusOK, gin.H{"position": updated})
}

func (h *PositionHandler) DeletePosition(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	if err := h.positionRepo.Delete(uint(posID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "position deleted"})
}

type ReorderInput struct {
	PositionIDs []uint `json:"position_ids" binding:"required"`
}

func (h *PositionHandler) ReorderPositions(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var input ReorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.positionRepo.Reorder(uint(jobID), input.PositionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reordered"})
}

type AssignDeviceInput struct {
	DeviceID  string `json:"device_id" binding:"required"`
	ScannedBy string `json:"scanned_by"`
}

func (h *PositionHandler) AssignDevice(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}

	var input AssignDeviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.positionRepo.AssignDevice(uint(posID), input.DeviceID, input.ScannedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "device assigned"})
}

func (h *PositionHandler) RemoveDevice(c *gin.Context) {
	posID, err := strconv.ParseUint(c.Param("posId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position ID"})
		return
	}
	deviceID := c.Param("devId")

	if err := h.positionRepo.RemoveDevice(uint(posID), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device removed"})
}

func (h *PositionHandler) GetPicklist(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetPicklist(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type PicklistItem struct {
		PositionID  uint     `json:"position_id"`
		ProductID   *uint    `json:"product_id"`
		ProductName string   `json:"product_name"`
		Needed      int      `json:"needed"`
		Scanned     int      `json:"scanned"`
		Remaining   int      `json:"remaining"`
		DeviceIDs   []string `json:"device_ids"`
	}

	items := make([]PicklistItem, 0, len(positions))
	for _, p := range positions {
		productName := ""
		if p.Product != nil {
			productName = p.Product.Name
		}
		needed := int(p.Quantity)
		scanned := len(p.Devices)
		deviceIDs := make([]string, 0, len(p.Devices))
		for _, d := range p.Devices {
			deviceIDs = append(deviceIDs, d.DeviceID)
		}
		items = append(items, PicklistItem{
			PositionID:  p.PositionID,
			ProductID:   p.ProductID,
			ProductName: productName,
			Needed:      needed,
			Scanned:     scanned,
			Remaining:   max(0, needed-scanned),
			DeviceIDs:   deviceIDs,
		})
	}

	c.JSON(http.StatusOK, gin.H{"picklist": items})
}

func (h *PositionHandler) GetTotals(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	positions, err := h.positionRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	eventDays := calcEventDays(job.StartDate, job.EndDate)

	subtotal := 0.0
	for _, p := range positions {
		dayFactor := 1.0
		if eventDays > 1 && p.FollowDayFactor > 0 {
			dayFactor = 1 + float64(eventDays-1)*p.FollowDayFactor
		}
		lineTotal := p.Quantity * p.UnitPrice * dayFactor
		discount := p.DiscountAmount + (lineTotal * p.DiscountPercent / 100)
		subtotal += lineTotal - discount
	}

	globalDiscount := 0.0
	if job.Discount > 0 {
		if job.DiscountType == "percent" {
			globalDiscount = subtotal * job.Discount / 100
		} else {
			globalDiscount = job.Discount
		}
	}

	netto := subtotal - globalDiscount
	taxRate := 19.0
	tax := netto * taxRate / 100
	brutto := netto + tax

	c.JSON(http.StatusOK, gin.H{
		"event_days":      eventDays,
		"subtotal":        math.Round(subtotal*100) / 100,
		"global_discount": math.Round(globalDiscount*100) / 100,
		"netto":           math.Round(netto*100) / 100,
		"tax_rate":        taxRate,
		"tax":             math.Round(tax*100) / 100,
		"brutto":          math.Round(brutto*100) / 100,
	})
}

func calcEventDays(start, end *time.Time) int {
	if start == nil || end == nil {
		return 1
	}
	days := int(end.Sub(*start).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}
