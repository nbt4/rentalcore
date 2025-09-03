package handlers

import (
	"net/http"

	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services"

	"github.com/gin-gonic/gin"
)

type BarcodeHandler struct {
	barcodeService *services.BarcodeService
	deviceRepo     *repository.DeviceRepository
}

func NewBarcodeHandler(barcodeService *services.BarcodeService, deviceRepo *repository.DeviceRepository) *BarcodeHandler {
	return &BarcodeHandler{
		barcodeService: barcodeService,
		deviceRepo:     deviceRepo,
	}
}

func (h *BarcodeHandler) GenerateDeviceQR(c *gin.Context) {
	serialNo := c.Param("serialNo")
	if serialNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Serial number is required"})
		return
	}

	// Verify device exists
	_, err := h.deviceRepo.GetBySerialNo(serialNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	qrBytes, err := h.barcodeService.GenerateDeviceQR(serialNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=device_"+serialNo+"_qr.png")
	c.Data(http.StatusOK, "image/png", qrBytes)
}

func (h *BarcodeHandler) GenerateDeviceBarcode(c *gin.Context) {
	serialNo := c.Param("serialNo")
	if serialNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Serial number is required"})
		return
	}

	// Verify device exists
	_, err := h.deviceRepo.GetBySerialNo(serialNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	barcodeBytes, err := h.barcodeService.GenerateDeviceBarcode(serialNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=device_"+serialNo+"_barcode.png")
	c.Data(http.StatusOK, "image/png", barcodeBytes)
}