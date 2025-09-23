package scan

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// DecodeRequest represents a server-side decode request
type DecodeRequest struct {
	ImageData string  `json:"imageData" binding:"required"` // Base64 encoded image
	Width     int     `json:"width" binding:"required"`
	Height    int     `json:"height" binding:"required"`
	ROI       *ROI    `json:"roi,omitempty"`
	Priority  int     `json:"priority"` // 0=auto, 1=1D, 2=2D
	Formats   []string `json:"formats,omitempty"`
}

// DecodeResponse represents a server-side decode response
type DecodeResponse struct {
	Success      bool      `json:"success"`
	Result       *Result   `json:"result,omitempty"`
	Error        string    `json:"error,omitempty"`
	ProcessingTime int64   `json:"processingTime"` // milliseconds
	Timestamp    int64     `json:"timestamp"`
	ServerDecode bool      `json:"serverDecode"` // Indicates this was server-side
}

// Result represents a decode result
type Result struct {
	Text         string  `json:"text"`
	Format       string  `json:"format"`
	CornerPoints []Point `json:"cornerPoints"`
	Confidence   float64 `json:"confidence"`
}

// Point represents a 2D coordinate
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ROI represents a region of interest
type ROI struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ServerDecoder handles server-side barcode decoding
type ServerDecoder struct {
	readers []gozxing.Reader
}

// NewServerDecoder creates a new server-side decoder
func NewServerDecoder() *ServerDecoder {
	readers := []gozxing.Reader{
		// 1D Readers
		oned.NewCode128Reader(),
		oned.NewCode39Reader(),
		oned.NewEAN13Reader(),
		oned.NewEAN8Reader(),
		oned.NewUPCAReader(),
		oned.NewUPCEReader(),
		oned.NewITFReader(),

		// 2D Readers
		qrcode.NewQRCodeReader(),
		// Note: DataMatrix and PDF417 not available in gozxing v0.1.1
	}

	return &ServerDecoder{
		readers: readers,
	}
}

// Decode processes a decode request
func (d *ServerDecoder) Decode(req *DecodeRequest) *DecodeResponse {
	startTime := time.Now()

	response := &DecodeResponse{
		Success:      false,
		Timestamp:    time.Now().UnixMilli(),
		ServerDecode: true,
	}

	// Decode base64 image data
	img, err := d.decodeImageData(req.ImageData)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to decode image: %v", err)
		response.ProcessingTime = time.Since(startTime).Milliseconds()
		return response
	}

	// Apply ROI if specified
	if req.ROI != nil {
		img, err = d.extractROI(img, req.ROI)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to extract ROI: %v", err)
			response.ProcessingTime = time.Since(startTime).Milliseconds()
			return response
		}
	}

	// Convert to binary bitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to create bitmap: %v", err)
		response.ProcessingTime = time.Since(startTime).Milliseconds()
		return response
	}

	// Get readers based on priority
	readersToTry := d.getReadersForPriority(req.Priority)

	// Try to decode with each reader
	hints := d.getDecodeHints(req.Priority, req.Formats)

	var result gozxing.Result
	var found bool

	for _, reader := range readersToTry {
		resultPtr, decodeErr := reader.Decode(bmp, hints)
		if decodeErr == nil && resultPtr != nil {
			result = *resultPtr
			found = true
			break
		}
	}

	response.ProcessingTime = time.Since(startTime).Milliseconds()

	if !found {
		response.Error = "No barcode found"
		return response
	}

	// Extract result data
	response.Success = true
	response.Result = &Result{
		Text:         result.GetText(),
		Format:       d.mapGozxingFormat(result.GetBarcodeFormat()),
		CornerPoints: d.extractCornerPoints(result),
		Confidence:   1.0, // gozxing doesn't provide confidence scores
	}

	return response
}

// decodeImageData decodes base64 image data
func (d *ServerDecoder) decodeImageData(imageData string) (image.Image, error) {
	// For now, assume PNG format
	// In production, you might want to support multiple formats
	// and auto-detect the format

	// Remove data URL prefix if present
	if len(imageData) > 22 && imageData[:22] == "data:image/png;base64," {
		imageData = imageData[22:]
	}

	// Decode base64
	data, err := d.base64Decode(imageData)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %v", err)
	}

	// Decode PNG
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("PNG decode failed: %v", err)
	}

	return img, nil
}

// base64Decode decodes a base64 string
func (d *ServerDecoder) base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// extractROI extracts a region of interest from the image
func (d *ServerDecoder) extractROI(img image.Image, roi *ROI) (image.Image, error) {
	bounds := img.Bounds()

	// Validate ROI bounds
	if roi.X < 0 || roi.Y < 0 ||
		roi.X+roi.Width > bounds.Max.X ||
		roi.Y+roi.Height > bounds.Max.Y {
		return nil, fmt.Errorf("ROI out of bounds")
	}

	// Create sub-image
	roiImg := image.NewRGBA(image.Rect(0, 0, roi.Width, roi.Height))

	for y := 0; y < roi.Height; y++ {
		for x := 0; x < roi.Width; x++ {
			roiImg.Set(x, y, img.At(roi.X+x, roi.Y+y))
		}
	}

	return roiImg, nil
}

// getReadersForPriority returns readers based on scan priority
func (d *ServerDecoder) getReadersForPriority(priority int) []gozxing.Reader {
	switch priority {
	case 1: // Priority1D
		return d.readers[:7] // First 7 are 1D readers
	case 2: // Priority2D
		return d.readers[7:] // Rest are 2D readers
	default: // Auto
		return d.readers
	}
}

// getDecodeHints returns decode hints
func (d *ServerDecoder) getDecodeHints(priority int, formats []string) map[gozxing.DecodeHintType]interface{} {
	hints := make(map[gozxing.DecodeHintType]interface{})

	// Set possible formats based on priority and requested formats
	var possibleFormats []gozxing.BarcodeFormat

	if len(formats) > 0 {
		// Use requested formats
		for _, format := range formats {
			if gzFormat := d.mapStringToGozxingFormat(format); gzFormat != -1 {
				possibleFormats = append(possibleFormats, gzFormat)
			}
		}
	} else {
		// Use priority-based formats
		switch priority {
		case 1: // Priority1D
			possibleFormats = []gozxing.BarcodeFormat{
				gozxing.BarcodeFormat_CODE_128,
				gozxing.BarcodeFormat_CODE_39,
				gozxing.BarcodeFormat_EAN_13,
				gozxing.BarcodeFormat_EAN_8,
				gozxing.BarcodeFormat_UPC_A,
				gozxing.BarcodeFormat_UPC_E,
				gozxing.BarcodeFormat_ITF,
			}
		case 2: // Priority2D
			possibleFormats = []gozxing.BarcodeFormat{
				gozxing.BarcodeFormat_QR_CODE,
			}
		default: // Auto
			possibleFormats = []gozxing.BarcodeFormat{
				gozxing.BarcodeFormat_CODE_128,
				gozxing.BarcodeFormat_CODE_39,
				gozxing.BarcodeFormat_EAN_13,
				gozxing.BarcodeFormat_EAN_8,
				gozxing.BarcodeFormat_UPC_A,
				gozxing.BarcodeFormat_UPC_E,
				gozxing.BarcodeFormat_ITF,
				gozxing.BarcodeFormat_QR_CODE,
			}
		}
	}

	if len(possibleFormats) > 0 {
		hints[gozxing.DecodeHintType_POSSIBLE_FORMATS] = possibleFormats
	}

	// Try harder for server-side decoding
	hints[gozxing.DecodeHintType_TRY_HARDER] = true

	return hints
}

// extractCornerPoints extracts corner points from decode result
func (d *ServerDecoder) extractCornerPoints(result gozxing.Result) []Point {
	resultPoints := result.GetResultPoints()
	if len(resultPoints) == 0 {
		return []Point{}
	}

	corners := make([]Point, len(resultPoints))
	for i, p := range resultPoints {
		corners[i] = Point{
			X: float64(p.GetX()),
			Y: float64(p.GetY()),
		}
	}

	return corners
}

// mapGozxingFormat maps gozxing format to string
func (d *ServerDecoder) mapGozxingFormat(format gozxing.BarcodeFormat) string {
	switch format {
	case gozxing.BarcodeFormat_CODE_128:
		return "CODE_128"
	case gozxing.BarcodeFormat_CODE_39:
		return "CODE_39"
	case gozxing.BarcodeFormat_EAN_13:
		return "EAN_13"
	case gozxing.BarcodeFormat_EAN_8:
		return "EAN_8"
	case gozxing.BarcodeFormat_UPC_A:
		return "UPC_A"
	case gozxing.BarcodeFormat_UPC_E:
		return "UPC_E"
	case gozxing.BarcodeFormat_ITF:
		return "ITF"
	case gozxing.BarcodeFormat_QR_CODE:
		return "QR_CODE"
	default:
		return "UNKNOWN"
	}
}

// mapStringToGozxingFormat maps string format to gozxing format
func (d *ServerDecoder) mapStringToGozxingFormat(format string) gozxing.BarcodeFormat {
	switch format {
	case "CODE_128":
		return gozxing.BarcodeFormat_CODE_128
	case "CODE_39":
		return gozxing.BarcodeFormat_CODE_39
	case "EAN_13":
		return gozxing.BarcodeFormat_EAN_13
	case "EAN_8":
		return gozxing.BarcodeFormat_EAN_8
	case "UPC_A":
		return gozxing.BarcodeFormat_UPC_A
	case "UPC_E":
		return gozxing.BarcodeFormat_UPC_E
	case "ITF":
		return gozxing.BarcodeFormat_ITF
	case "QR_CODE":
		return gozxing.BarcodeFormat_QR_CODE
	default:
		return -1 // Invalid format
	}
}