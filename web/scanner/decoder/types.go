package main

import (
	"errors"
)

// Supported barcode formats (currently available in gozxing v0.1.1)
type BarcodeFormat int

const (
	CODE128 BarcodeFormat = iota
	CODE39
	EAN13
	EAN8
	UPCA
	UPCE
	ITF
	QR_CODE
	// Note: DATA_MATRIX and PDF417 not available in current gozxing version
)

var formatNames = map[BarcodeFormat]string{
	CODE128: "CODE_128",
	CODE39:  "CODE_39",
	EAN13:   "EAN_13",
	EAN8:    "EAN_8",
	UPCA:    "UPC_A",
	UPCE:    "UPC_E",
	ITF:     "ITF",
	QR_CODE: "QR_CODE",
}

func (f BarcodeFormat) String() string {
	if name, ok := formatNames[f]; ok {
		return name
	}
	return "UNKNOWN"
}

// DecodeResult represents the result of a barcode decode operation
type DecodeResult struct {
	Text         string        `json:"text"`
	Format       string        `json:"format"`
	CornerPoints []Point       `json:"cornerPoints"`
	Confidence   float64       `json:"confidence"`
	Timestamp    int64         `json:"timestamp"`
}

// Point represents a 2D coordinate
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ROI represents a region of interest for focused scanning
type ROI struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ScanPriority indicates whether to prioritize 1D or 2D barcodes
type ScanPriority int

const (
	Priority1D ScanPriority = iota
	Priority2D
	PriorityAuto
)

// DecoderConfig holds configuration for the decoder
type DecoderConfig struct {
	EnabledFormats []BarcodeFormat `json:"enabledFormats"`
	Priority       ScanPriority    `json:"priority"`
	ROI            *ROI            `json:"roi,omitempty"`
	MaxRetries     int             `json:"maxRetries"`
	Timeout        int             `json:"timeout"` // milliseconds
}

// Standard errors
var (
	ErrNoCodeFound     = errors.New("no barcode found")
	ErrInvalidImage    = errors.New("invalid image data")
	ErrUnsupportedFmt  = errors.New("unsupported format")
	ErrTimeout         = errors.New("decode timeout")
	ErrInvalidROI      = errors.New("invalid ROI")
)

// Default configuration for industrial scanning
func DefaultConfig() DecoderConfig {
	return DecoderConfig{
		EnabledFormats: []BarcodeFormat{
			CODE128, CODE39, EAN13, EAN8, UPCA, UPCE, ITF, QR_CODE,
			// Note: DATA_MATRIX and PDF417 not available in current gozxing version
		},
		Priority:   PriorityAuto,
		MaxRetries: 3,
		Timeout:    100, // 100ms per decode attempt
	}
}