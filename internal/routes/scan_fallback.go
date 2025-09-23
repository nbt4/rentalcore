package routes

import (
	"net/http"

	"go-barcode-webapp/internal/scan"

	"github.com/gin-gonic/gin"
)

// ScanFallbackHandler handles server-side barcode decoding
type ScanFallbackHandler struct {
	decoder *scan.ServerDecoder
	enabled bool // Feature flag to enable/disable server-side decoding
}

// NewScanFallbackHandler creates a new scan fallback handler
func NewScanFallbackHandler() *ScanFallbackHandler {
	return &ScanFallbackHandler{
		decoder: scan.NewServerDecoder(),
		enabled: false, // Disabled by default - enable via environment variable
	}
}

// SetEnabled enables or disables the fallback decoder
func (h *ScanFallbackHandler) SetEnabled(enabled bool) {
	h.enabled = enabled
}

// IsEnabled returns whether the fallback decoder is enabled
func (h *ScanFallbackHandler) IsEnabled() bool {
	return h.enabled
}

// DecodeFallback handles server-side decode requests
func (h *ScanFallbackHandler) DecodeFallback(c *gin.Context) {
	// Check if feature is enabled
	if !h.enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "FEATURE_DISABLED",
			"message": "Server-side decode is disabled",
		})
		return
	}

	// Parse request
	var req scan.DecodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_REQUEST",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if req.Width <= 0 || req.Height <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_DIMENSIONS",
			"message": "Width and height must be positive",
		})
		return
	}

	if req.ImageData == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "MISSING_IMAGE_DATA",
			"message": "Image data is required",
		})
		return
	}

	// Process decode request
	response := h.decoder.Decode(&req)

	// Return appropriate HTTP status
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		// Return 422 for decode failures (valid request, but no barcode found)
		c.JSON(http.StatusUnprocessableEntity, response)
	}
}

// GetDecoderStatus returns the status of the fallback decoder
func (h *ScanFallbackHandler) GetDecoderStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":     h.enabled,
		"status":      "ready",
		"serverSide":  true,
		"supportedFormats": []string{
			"CODE_128", "CODE_39", "EAN_13", "EAN_8",
			"UPC_A", "UPC_E", "ITF", "QR_CODE",
		},
	})
}

// SetupScanFallbackRoutes sets up the fallback decode routes
func SetupScanFallbackRoutes(r *gin.Engine, handler *ScanFallbackHandler) {
	// API group for scan fallback
	api := r.Group("/api/scan")
	{
		// Decode endpoint
		api.POST("/decode", handler.DecodeFallback)

		// Status endpoint
		api.GET("/status", handler.GetDecoderStatus)
	}
}

// ScanFallbackMiddleware adds rate limiting for scan fallback requests
func ScanFallbackMiddleware() gin.HandlerFunc {
	// Simple rate limiting - in production you might want to use redis
	requestCounts := make(map[string]int)

	return gin.HandlerFunc(func(c *gin.Context) {
		// Rate limit by IP
		clientIP := c.ClientIP()

		// Simple rate limiting: max 60 requests per minute per IP
		if count, exists := requestCounts[clientIP]; exists && count > 60 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "RATE_LIMITED",
				"message": "Too many requests. Server-side decode is rate limited.",
			})
			c.Abort()
			return
		}

		requestCounts[clientIP]++

		// Reset counter periodically (simple implementation)
		// In production, use a proper rate limiter like golang.org/x/time/rate

		c.Next()
	})
}