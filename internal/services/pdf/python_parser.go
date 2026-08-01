package pdf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go-barcode-webapp/internal/logger"
)

// PythonParser wraps the Python OCR parser tool
type PythonParser struct {
	ParserPath string
	PythonPath string
	Timeout    time.Duration
}

// NewPythonParser creates a new Python parser instance
func NewPythonParser() *PythonParser {
	// Determine parser path relative to binary
	execPath, _ := os.Executable()
	basePath := filepath.Dir(execPath)
	parserPath := filepath.Join(basePath, "tools", "ocr_parser", "parser.py")

	// Check if parser exists, fallback to relative path for dev
	if _, err := os.Stat(parserPath); os.IsNotExist(err) {
		// Try relative path for development
		parserPath = "tools/ocr_parser/parser.py"
	}

	// Try to use virtualenv Python if available, otherwise use system Python
	pythonPath := "/opt/ocr-venv/bin/python3"
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		// Fall back to system Python (for development)
		pythonPath = "python3"
	}

	return &PythonParser{
		ParserPath: parserPath,
		PythonPath: pythonPath,
		Timeout:    10 * time.Second,
	}
}

// PythonParserInput represents the input structure for the Python parser
type PythonParserInput struct {
	RawText  string `json:"raw_text"`
	Language string `json:"language,omitempty"`
}

// PythonParserOutput represents the output structure from the Python parser
type PythonParserOutput struct {
	Document struct {
		Number         string  `json:"number,omitempty"`
		Date           string  `json:"date,omitempty"`
		CustomerName    string  `json:"customer_name,omitempty"`
		Subtotal        float64 `json:"subtotal,omitempty"`         // Subtotal before discount
		DiscountAmount  float64 `json:"discount_amount,omitempty"`  // Total discount
		DiscountPercent float64 `json:"discount_percent,omitempty"` // Discount percentage
		Total           float64 `json:"total,omitempty"`            // Final total after discount
	} `json:"document"`
	Items []struct {
		LineNumber      int     `json:"line_number"`
		Description     string  `json:"description"`
		Quantity        float64 `json:"quantity"`
		Unit            *string `json:"unit"`
		UnitPrice       float64 `json:"unit_price"`
		DiscountPercent float64 `json:"discount_percent"`
		LineTotal       float64 `json:"line_total"`
	} `json:"items"`
	Warnings []string `json:"warnings"`
}

// ParseDocument calls the Python parser and converts results to ParsedDocument
func (p *PythonParser) ParseDocument(rawText string) (*ParsedDocument, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	// Prepare input
	input := PythonParserInput{
		RawText:  rawText,
		Language: "de",
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %v", err)
	}

	// Execute Python parser
	cmd := exec.CommandContext(ctx, p.PythonPath, p.ParserPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.LogInfo("[PythonParser] Executing: %s %s", p.PythonPath, p.ParserPath)

	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("python parser timeout after %v", p.Timeout)
		}
		return nil, fmt.Errorf("python parser failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse output
	var output PythonParserOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse python output: %v\nOutput: %s", err, stdout.String())
	}

	// Log warnings
	if len(output.Warnings) > 0 {
		logger.LogInfo("[PythonParser] Warnings: %v", output.Warnings)
	}

	// Convert to ParsedDocument
	doc := &ParsedDocument{
		DocumentType:    DocTypeInvoice, // Default type
		CustomerName:    output.Document.CustomerName,
		DocumentNumber:  output.Document.Number,
		ParsedTotal:     output.Document.Subtotal,
		DiscountAmount:  output.Document.DiscountAmount,
		DiscountPercent: output.Document.DiscountPercent,
		TotalAmount:     output.Document.Total,
		Items:           make([]ParsedItem, 0, len(output.Items)),
		RawSections:     make(map[string]string),
		Metadata:        make(map[string]interface{}),
		ConfidenceScore: 0,
	}

	// Parse document date if provided
	if output.Document.Date != "" {
		if date, err := time.Parse("2006-01-02", output.Document.Date); err == nil {
			doc.DocumentDate = date
		}
	}

	// Convert items
	for _, pyItem := range output.Items {
		item := ParsedItem{
			LineNumber:      pyItem.LineNumber,
			RawText:         pyItem.Description,
			DetectedType:    ItemTypeProduct,
			ProductName:     pyItem.Description,
			Quantity:        int(pyItem.Quantity),
			UnitPrice:       pyItem.UnitPrice,
			LineTotal:       pyItem.LineTotal,
			ConfidenceScore: 90.0, // High confidence from Python parser
		}
		doc.Items = append(doc.Items, item)
	}

	// Calculate confidence score
	doc.ConfidenceScore = p.calculateConfidence(doc)

	// Store metadata
	doc.Metadata["parser_version"] = "python_v1"
	doc.Metadata["warnings"] = output.Warnings
	doc.Metadata["item_count"] = len(output.Items)

	logger.LogInfo("[PythonParser] Parsed successfully: items=%d, confidence=%.2f", len(doc.Items), doc.ConfidenceScore)

	return doc, nil
}

// calculateConfidence calculates overall confidence score
func (p *PythonParser) calculateConfidence(doc *ParsedDocument) float64 {
	score := 0.0
	maxScore := 7.0

	if doc.DocumentType != DocTypeUnknown {
		score += 1.0
	}
	if doc.CustomerName != "" {
		score += 1.0
	}
	if doc.DocumentNumber != "" {
		score += 1.5
	}
	if !doc.DocumentDate.IsZero() {
		score += 1.0
	}
	if doc.TotalAmount > 0 {
		score += 1.0
	}
	if len(doc.Items) > 0 {
		score += 1.5
	}

	return (score / maxScore) * 100.0
}

// IsAvailable checks if the Python parser is available
func (p *PythonParser) IsAvailable() bool {
	// Check if Python is available
	if _, err := exec.LookPath(p.PythonPath); err != nil {
		logger.LogInfo("[PythonParser] Python not found in PATH: %v", err)
		return false
	}

	// Check if parser script exists
	if _, err := os.Stat(p.ParserPath); os.IsNotExist(err) {
		logger.LogInfo("[PythonParser] Parser script not found at: %s", p.ParserPath)
		return false
	}

	// Verify the runtime can import the parser's direct dependency. Checking the
	// executable alone misses broken or incomplete copied virtual environments.
	cmd := exec.Command(p.PythonPath, "-c", "import click")
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.LogInfo("[PythonParser] Python environment is incomplete: %v: %s", err, strings.TrimSpace(string(output)))
		return false
	}

	return true
}
