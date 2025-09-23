package main

import (
	"syscall/js"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
	"github.com/makiuchi-d/gozxing/qrcode"
)

var (
	globalCache   *DedupeCache
	globalReaders []gozxing.Reader
	isInitialized bool
)

// initializeDecoder sets up the decoder with optimized readers
func initializeDecoder() {
	if isInitialized {
		return
	}

	// Create dedupe cache with 1.5 second cooldown
	globalCache = NewDedupeCache(1500 * time.Millisecond)

	// Initialize readers with our target formats
	globalReaders = []gozxing.Reader{
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

	isInitialized = true
}

// Decode processes an RGBA frame and returns barcode results
func Decode(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return map[string]interface{}{
			"success": false,
			"error":   "insufficient arguments: need frameData, width, height",
		}
	}

	// Initialize decoder if needed
	initializeDecoder()

	// Extract arguments
	frameDataJS := args[0]
	width := args[1].Int()
	height := args[2].Int()

	// Optional ROI parameter
	var roi *ROI
	if len(args) > 3 && !args[3].IsNull() {
		roiJS := args[3]
		roi = &ROI{
			X:      roiJS.Get("x").Int(),
			Y:      roiJS.Get("y").Int(),
			Width:  roiJS.Get("width").Int(),
			Height: roiJS.Get("height").Int(),
		}
	}

	// Optional priority parameter
	priority := PriorityAuto
	if len(args) > 4 && !args[4].IsNull() {
		priority = ScanPriority(args[4].Int())
	}

	// Convert JS Uint8Array to Go byte slice
	frameData := make([]byte, frameDataJS.Get("length").Int())
	js.CopyBytesToGo(frameData, frameDataJS)

	// Decode the frame
	result, err := decodeFrame(frameData, width, height, roi, priority)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	// Check for duplicates
	if globalCache.IsDuplicate(*result) {
		return map[string]interface{}{
			"success":   false,
			"duplicate": true,
			"error":     "duplicate barcode (cooldown active)",
		}
	}

	// Add to cache and return success
	globalCache.Add(*result)

	return map[string]interface{}{
		"success": true,
		"result":  resultToJS(*result),
	}
}

// decodeFrame performs the actual barcode decoding
func decodeFrame(frameData []byte, width, height int, roi *ROI, priority ScanPriority) (*DecodeResult, error) {
	// Convert RGBA data to image
	img, err := rgbaToImage(frameData, width, height)
	if err != nil {
		return nil, err
	}

	// Apply ROI if specified
	if roi != nil {
		img, err = extractROI(img, roi)
		if err != nil {
			return nil, err
		}
	} else if priority == Priority1D {
		// Create center ROI for 1D priority
		centerROI := createCenterROI(width, height, 0.7)
		img, err = extractROI(img, centerROI)
		if err == nil {
			// ROI extraction successful, adjust width/height for corner point calculation
			width = centerROI.Width
			height = centerROI.Height
		}
		// If ROI extraction fails, continue with full image
	}

	// Preprocess image for better recognition
	img = preprocessImage(img)

	// Convert to gozxing BinaryBitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, err
	}

	// Try each reader until we find a match
	hints := getDecodeHints(priority)

	var result gozxing.Result
	var found bool

	// Try readers based on priority
	readersToTry := getReadersForPriority(priority)

	for _, reader := range readersToTry {
		resultPtr, decodeErr := reader.Decode(bmp, hints)
		if decodeErr == nil && resultPtr != nil {
			result = *resultPtr
			found = true
			break
		}
	}

	if !found {
		return nil, ErrNoCodeFound
	}

	// Extract corner points if available
	cornerPoints := extractCornerPoints(result, width, height)

	// Determine format
	format := mapGozxingFormat(result.GetBarcodeFormat())

	return &DecodeResult{
		Text:         result.GetText(),
		Format:       format,
		CornerPoints: cornerPoints,
		Confidence:   1.0, // gozxing doesn't provide confidence scores
		Timestamp:    time.Now().UnixMilli(),
	}, nil
}

// getReadersForPriority returns readers based on scan priority
func getReadersForPriority(priority ScanPriority) []gozxing.Reader {
	switch priority {
	case Priority1D:
		// Return only 1D readers
		return globalReaders[:7] // First 7 are 1D readers
	case Priority2D:
		// Return only 2D readers
		return globalReaders[7:] // Rest are 2D readers
	default:
		// Return all readers (auto mode)
		return globalReaders
	}
}

// getDecodeHints returns decode hints based on scan priority
func getDecodeHints(priority ScanPriority) map[gozxing.DecodeHintType]interface{} {
	hints := make(map[gozxing.DecodeHintType]interface{})

	switch priority {
	case Priority1D:
		// Enable 1D format readers only
		hints[gozxing.DecodeHintType_POSSIBLE_FORMATS] = []gozxing.BarcodeFormat{
			gozxing.BarcodeFormat_CODE_128,
			gozxing.BarcodeFormat_CODE_39,
			gozxing.BarcodeFormat_EAN_13,
			gozxing.BarcodeFormat_EAN_8,
			gozxing.BarcodeFormat_UPC_A,
			gozxing.BarcodeFormat_UPC_E,
			gozxing.BarcodeFormat_ITF,
		}
	case Priority2D:
		// Enable 2D format readers only
		hints[gozxing.DecodeHintType_POSSIBLE_FORMATS] = []gozxing.BarcodeFormat{
			gozxing.BarcodeFormat_QR_CODE,
			// Note: DataMatrix and PDF417 not available in gozxing v0.1.1
		}
	default:
		// Auto mode - try all supported formats
		hints[gozxing.DecodeHintType_POSSIBLE_FORMATS] = []gozxing.BarcodeFormat{
			gozxing.BarcodeFormat_CODE_128,
			gozxing.BarcodeFormat_CODE_39,
			gozxing.BarcodeFormat_EAN_13,
			gozxing.BarcodeFormat_EAN_8,
			gozxing.BarcodeFormat_UPC_A,
			gozxing.BarcodeFormat_UPC_E,
			gozxing.BarcodeFormat_ITF,
			gozxing.BarcodeFormat_QR_CODE,
			// Note: DataMatrix and PDF417 not available in gozxing v0.1.1
		}
	}

	// Try harder for industrial environments
	hints[gozxing.DecodeHintType_TRY_HARDER] = true

	return hints
}

// extractCornerPoints extracts corner points from decode result
func extractCornerPoints(result gozxing.Result, imgWidth, imgHeight int) []Point {
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

// mapGozxingFormat maps gozxing format to our string format
func mapGozxingFormat(format gozxing.BarcodeFormat) string {
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
	// Note: DataMatrix and PDF417 not available in gozxing v0.1.1
	default:
		return "UNKNOWN"
	}
}

// resultToJS converts DecodeResult to JS-compatible object
func resultToJS(result DecodeResult) map[string]interface{} {
	corners := make([]map[string]interface{}, len(result.CornerPoints))
	for i, point := range result.CornerPoints {
		corners[i] = map[string]interface{}{
			"x": point.X,
			"y": point.Y,
		}
	}

	return map[string]interface{}{
		"text":         result.Text,
		"format":       result.Format,
		"cornerPoints": corners,
		"confidence":   result.Confidence,
		"timestamp":    result.Timestamp,
	}
}

// GetCacheStats returns dedupe cache statistics
func GetCacheStats(this js.Value, args []js.Value) interface{} {
	if globalCache == nil {
		return map[string]interface{}{"error": "cache not initialized"}
	}
	return globalCache.GetStats()
}

// ClearCache clears the dedupe cache
func ClearCache(this js.Value, args []js.Value) interface{} {
	if globalCache == nil {
		initializeDecoder()
	}
	globalCache.Clear()
	return map[string]interface{}{"success": true}
}

// main function required for WASM
func main() {
	// Keep the Go program running
	c := make(chan struct{}, 0)

	// Export functions to JavaScript
	js.Global().Set("goDecode", js.FuncOf(Decode))
	js.Global().Set("goGetCacheStats", js.FuncOf(GetCacheStats))
	js.Global().Set("goClearCache", js.FuncOf(ClearCache))

	// Signal that the WASM module is ready
	js.Global().Set("goWasmReady", js.ValueOf(true))

	<-c
}