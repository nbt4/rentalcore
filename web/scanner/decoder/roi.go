package main

import (
	"image"
	"image/color"
)

// extractROI extracts a region of interest from the image
func extractROI(img image.Image, roi *ROI) (image.Image, error) {
	if roi == nil {
		return img, nil
	}

	bounds := img.Bounds()

	// Validate ROI bounds
	if roi.X < 0 || roi.Y < 0 ||
		roi.X+roi.Width > bounds.Max.X ||
		roi.Y+roi.Height > bounds.Max.Y {
		return nil, ErrInvalidROI
	}

	// Create a new image with the ROI
	roiImg := image.NewRGBA(image.Rect(0, 0, roi.Width, roi.Height))

	for y := 0; y < roi.Height; y++ {
		for x := 0; x < roi.Width; x++ {
			roiImg.Set(x, y, img.At(roi.X+x, roi.Y+y))
		}
	}

	return roiImg, nil
}

// createCenterROI creates a center ROI for 1D barcode priority scanning
func createCenterROI(width, height int, percentage float64) *ROI {
	if percentage <= 0 || percentage >= 1 {
		percentage = 0.7 // Default 70% center area
	}

	roiWidth := int(float64(width) * percentage)
	roiHeight := int(float64(height) * percentage)

	x := (width - roiWidth) / 2
	y := (height - roiHeight) / 2

	return &ROI{
		X:      x,
		Y:      y,
		Width:  roiWidth,
		Height: roiHeight,
	}
}

// preprocessImage applies basic image preprocessing for better decode rates
func preprocessImage(img image.Image) image.Image {
	bounds := img.Bounds()
	processed := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(originalColor).(color.Gray)
			processed.Set(x, y, grayColor)
		}
	}

	return processed
}

// rgbaToImage converts RGBA byte slice to image.Image
func rgbaToImage(data []byte, width, height int) (image.Image, error) {
	if len(data) != width*height*4 {
		return nil, ErrInvalidImage
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 4
			r := data[idx]
			g := data[idx+1]
			b := data[idx+2]
			a := data[idx+3]

			img.Set(x, y, color.RGBA{r, g, b, a})
		}
	}

	return img, nil
}