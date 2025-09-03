package services

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/skip2/go-qrcode"
)

type BarcodeService struct{}

func NewBarcodeService() *BarcodeService {
	return &BarcodeService{}
}

func (s *BarcodeService) GenerateQRCode(data string, size int) ([]byte, error) {
	pngBytes, err := qrcode.Encode(data, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	return pngBytes, nil
}

func (s *BarcodeService) GenerateBarcode(data string) ([]byte, error) {
	// Create Code128 barcode
	bc, err := code128.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode: %w", err)
	}

	// Scale the barcode to reasonable size
	scaledBC, err := barcode.Scale(bc, 200, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, scaledBC)
	if err != nil {
		return nil, fmt.Errorf("failed to encode barcode as PNG: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *BarcodeService) GenerateDeviceQR(deviceID string) ([]byte, error) {
	data := fmt.Sprintf("DEVICE:%s", deviceID)
	return s.GenerateQRCode(data, 256)
}

func (s *BarcodeService) GenerateDeviceBarcode(deviceID string) ([]byte, error) {
	return s.GenerateBarcode(deviceID)
}