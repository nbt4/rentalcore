package pdf

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"

	"go-barcode-webapp/internal/logger"
)

// OCREngine handles text extraction from PDFs using ledongthuc/pdf
type OCREngine struct {
	TempDir  string
	Language string // Placeholder for future OCR support
}

// NewOCREngine creates a new OCR engine instance
func NewOCREngine(tempDir string) *OCREngine {
	return &OCREngine{
		TempDir:  tempDir,
		Language: "eng+deu", // English + German (for future use)
	}
}

// OCRResult represents the result of text extraction
type OCRResult struct {
	Text           string
	Confidence     float64
	PageCount      int
	Method         string // "text_based", "ocr", or "hybrid"
	PageTexts      []string
	PageConfidence []float64
}

// ExtractTextWithOCR extracts text from a PDF
// Currently uses text-based extraction only (ledongthuc/pdf)
// OCR support can be added later if needed
func (o *OCREngine) ExtractTextWithOCR(pdfPath string) (*OCRResult, error) {
	logger.LogInfo("Extracting text from PDF: %s", pdfPath)

	// Use ledongthuc/pdf for text extraction
	file, reader, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %v", err)
	}
	defer file.Close()

	var textBuilder strings.Builder
	var pageTexts []string
	numPages := reader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			pageTexts = append(pageTexts, "")
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			logger.LogInfo("Warning: failed to extract text from page %d: %v", pageNum, err)
			pageTexts = append(pageTexts, "")
			continue
		}

		pageTexts = append(pageTexts, text)
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	extractedText := textBuilder.String()

	// Calculate confidence based on text extraction success
	confidence := 95.0 // High confidence for text-based extraction
	if len(strings.TrimSpace(extractedText)) < 50 {
		confidence = 60.0 // Lower confidence if minimal text found
	}

	logger.LogInfo("Text extraction complete: %d pages, %d characters", numPages, len(extractedText))

	return &OCRResult{
		Text:           extractedText,
		Confidence:     confidence,
		PageCount:      numPages,
		Method:         "text_based",
		PageTexts:      pageTexts,
		PageConfidence: make([]float64, numPages), // All pages get same confidence
	}, nil
}
