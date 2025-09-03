package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"

	"github.com/jung-kurt/gofpdf"
)

type PDFServiceNew struct {
	tempDir   string
	pdfConfig *config.PDFConfig
}

func NewPDFServiceNew(pdfConfig *config.PDFConfig) *PDFServiceNew {
	tempDir := filepath.Join(os.TempDir(), "rentalcore_pdfs")
	
	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("Warning: Could not create PDF temp directory: %v", err)
		tempDir = os.TempDir()
	}

	return &PDFServiceNew{
		tempDir:   tempDir,
		pdfConfig: pdfConfig,
	}
}

// GenerateInvoicePDF generates a PDF from an invoice with robust error handling
func (s *PDFServiceNew) GenerateInvoicePDF(invoice *models.Invoice, company *models.CompanySettings, settings *models.InvoiceSettings) ([]byte, error) {
	log.Printf("PDFServiceNew: Generating PDF for invoice %s", invoice.InvoiceNumber)

	// Validate inputs
	if invoice == nil {
		return nil, fmt.Errorf("invoice cannot be nil")
	}
	if company == nil {
		company = s.getDefaultCompanySettings()
	}
	if settings == nil {
		settings = s.getDefaultInvoiceSettings()
	}

	// Try multiple PDF generation methods in order of preference
	methods := []struct {
		name string
		fn   func(*models.Invoice, *models.CompanySettings, *models.InvoiceSettings) ([]byte, error)
	}{
		{"Chrome/Chromium", s.generateWithChrome},
		{"wkhtmltopdf", s.generateWithWKHTMLToPDF},
		{"gofpdf", s.generateWithGofpdf},
	}

	var lastErr error
	for _, method := range methods {
		pdfBytes, err := method.fn(invoice, company, settings)
		if err == nil && len(pdfBytes) > 0 {
			// Validate that it's actually PDF content
			if len(pdfBytes) >= 4 && string(pdfBytes[:4]) == "%PDF" {
				log.Printf("PDFServiceNew: Successfully generated PDF using %s (%d bytes)", method.name, len(pdfBytes))
				return pdfBytes, nil
			} else {
				log.Printf("PDFServiceNew: %s returned invalid PDF content, trying next method", method.name)
				lastErr = fmt.Errorf("%s returned invalid PDF content", method.name)
				continue
			}
		}
		lastErr = err
		log.Printf("PDFServiceNew: %s failed: %v", method.name, err)
	}

	return nil, fmt.Errorf("all PDF generation methods failed, last error: %v", lastErr)
}

// generateWithChrome uses Chrome/Chromium headless for PDF generation
func (s *PDFServiceNew) generateWithChrome(invoice *models.Invoice, company *models.CompanySettings, settings *models.InvoiceSettings) ([]byte, error) {
	// Check for Chrome/Chromium
	chromePaths := []string{"google-chrome", "chromium", "chromium-browser", "chrome"}
	var chromePath string

	for _, path := range chromePaths {
		if _, err := exec.LookPath(path); err == nil {
			chromePath = path
			break
		}
	}

	if chromePath == "" {
		return nil, fmt.Errorf("Chrome/Chromium not found")
	}

	// Generate HTML content
	htmlContent, err := s.generateInvoiceHTML(invoice, company, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTML: %v", err)
	}

	// Create temporary files
	htmlFile := filepath.Join(s.tempDir, fmt.Sprintf("invoice_%s_%d.html", invoice.InvoiceNumber, time.Now().UnixNano()))
	pdfFile := filepath.Join(s.tempDir, fmt.Sprintf("invoice_%s_%d.pdf", invoice.InvoiceNumber, time.Now().UnixNano()))

	// Cleanup files
	defer func() {
		os.Remove(htmlFile)
		os.Remove(pdfFile)
	}()

	// Write HTML file
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write HTML file: %v", err)
	}

	// Execute Chrome headless
	cmd := exec.Command(chromePath,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-software-rasterizer",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--print-to-pdf="+pdfFile,
		"--print-to-pdf-no-header",
		"--virtual-time-budget=5000",
		"file://"+htmlFile)

	// Set timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Chrome PDF generation failed: %v", err)
	}

	// Check if PDF file exists and has content
	if _, err := os.Stat(pdfFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("PDF file was not generated")
	}

	// Read PDF file
	pdfBytes, err := os.ReadFile(pdfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %v", err)
	}

	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("generated PDF file is empty")
	}

	return pdfBytes, nil
}

// generateWithWKHTMLToPDF uses wkhtmltopdf for PDF generation
func (s *PDFServiceNew) generateWithWKHTMLToPDF(invoice *models.Invoice, company *models.CompanySettings, settings *models.InvoiceSettings) ([]byte, error) {
	// Check if wkhtmltopdf is available
	if _, err := exec.LookPath("wkhtmltopdf"); err != nil {
		return nil, fmt.Errorf("wkhtmltopdf not found")
	}

	// Generate HTML content
	htmlContent, err := s.generateInvoiceHTML(invoice, company, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTML: %v", err)
	}

	// Create temporary files
	htmlFile := filepath.Join(s.tempDir, fmt.Sprintf("invoice_%s_%d.html", invoice.InvoiceNumber, time.Now().UnixNano()))
	pdfFile := filepath.Join(s.tempDir, fmt.Sprintf("invoice_%s_%d.pdf", invoice.InvoiceNumber, time.Now().UnixNano()))

	// Cleanup files
	defer func() {
		os.Remove(htmlFile)
		os.Remove(pdfFile)
	}()

	// Write HTML file
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write HTML file: %v", err)
	}

	// Execute wkhtmltopdf
	cmd := exec.Command("wkhtmltopdf",
		"--page-size", "A4",
		"--margin-top", "1cm",
		"--margin-bottom", "1cm",
		"--margin-left", "1cm",
		"--margin-right", "1cm",
		"--enable-local-file-access",
		"--disable-smart-shrinking",
		"--print-media-type",
		htmlFile, pdfFile)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wkhtmltopdf failed: %v", err)
	}

	// Check if PDF file exists
	if _, err := os.Stat(pdfFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("PDF file was not generated")
	}

	// Read PDF file
	pdfBytes, err := os.ReadFile(pdfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %v", err)
	}

	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("generated PDF file is empty")
	}

	return pdfBytes, nil
}

// generateWithGofpdf creates a PDF using the gofpdf library (fallback)
func (s *PDFServiceNew) generateWithGofpdf(invoice *models.Invoice, company *models.CompanySettings, settings *models.InvoiceSettings) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)
	
	// Header with company and invoice info
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(37, 99, 235) // Blue color
	pdf.Cell(0, 10, company.CompanyName)
	pdf.Ln(8)
	
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0) // Black
	if company.AddressLine1 != nil {
		pdf.Cell(0, 5, *company.AddressLine1)
		pdf.Ln(5)
	}
	if company.City != nil || company.PostalCode != nil {
		address := ""
		if company.PostalCode != nil {
			address += *company.PostalCode + " "
		}
		if company.City != nil {
			address += *company.City
		}
		if address != "" {
			pdf.Cell(0, 5, address)
			pdf.Ln(5)
		}
	}
	if company.Phone != nil {
		pdf.Cell(0, 5, "Phone: "+*company.Phone)
		pdf.Ln(5)
	}
	if company.Email != nil {
		pdf.Cell(0, 5, "Email: "+*company.Email)
		pdf.Ln(5)
	}
	
	pdf.Ln(10)
	
	// Invoice title and details
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(37, 99, 235)
	pdf.Cell(0, 15, "INVOICE")
	pdf.Ln(15)
	
	// Invoice metadata in table format
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(248, 249, 250)
	
	// Two column layout for invoice details
	colWidth := 40.0
	
	pdf.CellFormat(colWidth, 8, "Invoice #:", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidth, 8, invoice.InvoiceNumber, "1", 1, "", false, 0, "")
	
	pdf.CellFormat(colWidth, 8, "Issue Date:", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidth, 8, invoice.IssueDate.Format("02.01.2006"), "1", 1, "", false, 0, "")
	
	pdf.CellFormat(colWidth, 8, "Due Date:", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidth, 8, invoice.DueDate.Format("02.01.2006"), "1", 1, "", false, 0, "")
	
	pdf.CellFormat(colWidth, 8, "Status:", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidth, 8, strings.ToUpper(invoice.Status), "1", 1, "", false, 0, "")
	
	pdf.Ln(10)

	// Customer information
	if invoice.Customer != nil {
		pdf.SetFont("Arial", "B", 12)
		pdf.SetTextColor(37, 99, 235)
		pdf.Cell(0, 8, "Bill To:")
		pdf.Ln(8)
		
		pdf.SetFont("Arial", "", 10)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFillColor(248, 249, 250)
		
		pdf.Rect(20, pdf.GetY(), 80, 25, "F")
		pdf.CellFormat(80, 6, invoice.Customer.GetDisplayName(), "", 1, "", false, 0, "")
		
		if invoice.Customer.Email != nil {
			pdf.CellFormat(80, 5, *invoice.Customer.Email, "", 1, "", false, 0, "")
		}
		if invoice.Customer.PhoneNumber != nil {
			pdf.CellFormat(80, 5, *invoice.Customer.PhoneNumber, "", 1, "", false, 0, "")
		}
		if invoice.Customer.Street != nil && invoice.Customer.HouseNumber != nil {
			pdf.CellFormat(80, 5, *invoice.Customer.Street+" "+*invoice.Customer.HouseNumber, "", 1, "", false, 0, "")
		}
		if invoice.Customer.ZIP != nil && invoice.Customer.City != nil {
			pdf.CellFormat(80, 5, *invoice.Customer.ZIP+" "+*invoice.Customer.City, "", 1, "", false, 0, "")
		}
	}
	
	pdf.Ln(15)

	// Line items table
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(37, 99, 235)
	
	pdf.CellFormat(90, 10, "Description", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 10, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Unit Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Total", "1", 1, "C", true, 0, "")

	// Line items
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(248, 249, 250)
	
	fill := false
	for _, item := range invoice.LineItems {
		pdf.CellFormat(90, 8, item.Description, "1", 0, "", fill, 0, "")
		pdf.CellFormat(20, 8, fmt.Sprintf("%.1f", item.Quantity), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%s%.2f", settings.CurrencySymbol, item.UnitPrice), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%s%.2f", settings.CurrencySymbol, item.TotalPrice), "1", 1, "R", fill, 0, "")
		fill = !fill
	}

	pdf.Ln(8)

	// Totals section
	pdf.SetFont("Arial", "B", 10)
	totalsX := 120.0
	pdf.SetX(totalsX)
	
	pdf.CellFormat(30, 8, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%s%.2f", settings.CurrencySymbol, invoice.Subtotal), "", 1, "R", false, 0, "")
	
	pdf.SetX(totalsX)
	pdf.CellFormat(30, 8, fmt.Sprintf("Tax (%.1f%%):", invoice.TaxRate), "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%s%.2f", settings.CurrencySymbol, invoice.TaxAmount), "", 1, "R", false, 0, "")
	
	if invoice.DiscountAmount > 0 {
		pdf.SetX(totalsX)
		pdf.CellFormat(30, 8, "Discount:", "", 0, "R", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("-%s%.2f", settings.CurrencySymbol, invoice.DiscountAmount), "", 1, "R", false, 0, "")
	}
	
	// Total with background
	pdf.SetX(totalsX)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(37, 99, 235)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(30, 10, "TOTAL:", "1", 0, "R", true, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%s%.2f", settings.CurrencySymbol, invoice.TotalAmount), "1", 1, "R", true, 0, "")

	// Notes section
	if invoice.Notes != nil && *invoice.Notes != "" {
		pdf.Ln(15)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetTextColor(37, 99, 235)
		pdf.Cell(0, 8, "Notes:")
		pdf.Ln(8)
		
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFillColor(248, 249, 250)
		
		// Create a background for notes
		notesHeight := float64(len(strings.Split(*invoice.Notes, "\n")) * 5 + 10)
		pdf.Rect(20, pdf.GetY(), 170, notesHeight, "F")
		
		lines := strings.Split(*invoice.Notes, "\n")
		for _, line := range lines {
			if len(line) > 80 {
				// Word wrap long lines
				words := strings.Fields(line)
				currentLine := ""
				for _, word := range words {
					if len(currentLine)+len(word)+1 > 80 {
						pdf.Cell(0, 5, currentLine)
						pdf.Ln(5)
						currentLine = word
					} else {
						if currentLine != "" {
							currentLine += " "
						}
						currentLine += word
					}
				}
				if currentLine != "" {
					pdf.Cell(0, 5, currentLine)
					pdf.Ln(5)
				}
			} else {
				pdf.Cell(0, 5, line)
				pdf.Ln(5)
			}
		}
	}
	
	// Footer
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	footerText := fmt.Sprintf("Generated on %s", time.Now().Format("02.01.2006 15:04:05"))
	if company.TaxNumber != nil {
		footerText += fmt.Sprintf(" | Tax Number: %s", *company.TaxNumber)
	}
	pdf.Cell(0, 5, footerText)

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF with gofpdf: %v", err)
	}

	pdfBytes := buf.Bytes()
	
	// Validate PDF output
	if len(pdfBytes) < 4 || string(pdfBytes[:4]) != "%PDF" {
		return nil, fmt.Errorf("gofpdf did not generate valid PDF content")
	}

	return pdfBytes, nil
}

// generateInvoiceHTML creates clean, professional HTML for the invoice
func (s *PDFServiceNew) generateInvoiceHTML(invoice *models.Invoice, company *models.CompanySettings, settings *models.InvoiceSettings) (string, error) {
	tmplContent := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Invoice {{.Invoice.InvoiceNumber}}</title>
    <style>
        @page {
            size: A4;
            margin: 1cm;
        }
        
        body {
            font-family: Arial, sans-serif;
            font-size: 12px;
            line-height: 1.4;
            color: #333;
            margin: 0;
            padding: 0;
        }
        
        .invoice-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 30px;
            border-bottom: 2px solid #2563eb;
            padding-bottom: 20px;
        }
        
        .company-info h1 {
            margin: 0 0 10px 0;
            font-size: 24px;
            color: #2563eb;
        }
        
        .company-info div {
            margin: 2px 0;
        }
        
        .invoice-details {
            text-align: right;
        }
        
        .invoice-title {
            font-size: 28px;
            color: #2563eb;
            margin-bottom: 10px;
            font-weight: bold;
        }
        
        .invoice-meta table {
            margin-left: auto;
            border-collapse: collapse;
        }
        
        .invoice-meta td {
            padding: 4px 8px;
            border: 1px solid #ddd;
        }
        
        .invoice-meta td:first-child {
            font-weight: bold;
            background-color: #f8f9fa;
        }
        
        .billing-section {
            display: flex;
            justify-content: space-between;
            margin-bottom: 30px;
        }
        
        .bill-to, .job-info {
            flex: 1;
            margin-right: 20px;
        }
        
        .bill-to h3, .job-info h3 {
            margin-bottom: 10px;
            color: #2563eb;
            font-size: 14px;
        }
        
        .address-box {
            border: 1px solid #ddd;
            padding: 15px;
            background-color: #f8f9fa;
        }
        
        .items-table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
        }
        
        .items-table th {
            background-color: #2563eb;
            color: white;
            padding: 10px;
            text-align: left;
            font-weight: bold;
        }
        
        .items-table td {
            padding: 8px 10px;
            border-bottom: 1px solid #ddd;
        }
        
        .items-table tbody tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        
        .text-right {
            text-align: right;
        }
        
        .totals-section {
            display: flex;
            justify-content: space-between;
            margin-bottom: 30px;
        }
        
        .notes {
            flex: 1;
            margin-right: 30px;
        }
        
        .totals {
            flex: 0 0 300px;
        }
        
        .totals-table {
            width: 100%;
            border-collapse: collapse;
        }
        
        .totals-table td {
            padding: 8px 10px;
            border-bottom: 1px solid #ddd;
        }
        
        .totals-table .total-row {
            font-weight: bold;
            font-size: 14px;
            background-color: #2563eb;
            color: white;
        }
        
        .status-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: bold;
            text-transform: uppercase;
        }
        
        .status-draft { background-color: #6c757d; color: white; }
        .status-sent { background-color: #17a2b8; color: white; }
        .status-paid { background-color: #28a745; color: white; }
        .status-overdue { background-color: #dc3545; color: white; }
        .status-cancelled { background-color: #6c757d; color: white; }
        
        .footer-info {
            border-top: 1px solid #ddd;
            padding-top: 20px;
            text-align: center;
            font-size: 11px;
            color: #666;
            margin-top: 30px;
        }
    </style>
</head>
<body>
    <!-- Invoice Header -->
    <div class="invoice-header">
        <div class="company-info">
            <h1>{{.Company.CompanyName}}</h1>
            {{if .Company.AddressLine1}}<div>{{.Company.AddressLine1}}</div>{{end}}
            {{if .Company.AddressLine2}}<div>{{.Company.AddressLine2}}</div>{{end}}
            {{if or .Company.City .Company.PostalCode}}
            <div>
                {{if .Company.PostalCode}}{{.Company.PostalCode}} {{end}}{{.Company.City}}
                {{if .Company.State}}, {{.Company.State}}{{end}}
            </div>
            {{end}}
            {{if .Company.Country}}<div>{{.Company.Country}}</div>{{end}}
            {{if .Company.Phone}}<div><strong>Phone:</strong> {{.Company.Phone}}</div>{{end}}
            {{if .Company.Email}}<div><strong>Email:</strong> {{.Company.Email}}</div>{{end}}
        </div>
        
        <div class="invoice-details">
            <div class="invoice-title">INVOICE</div>
            <div class="invoice-meta">
                <table>
                    <tr>
                        <td>Invoice #:</td>
                        <td>{{.Invoice.InvoiceNumber}}</td>
                    </tr>
                    <tr>
                        <td>Issue Date:</td>
                        <td>{{.Invoice.IssueDate.Format "02.01.2006"}}</td>
                    </tr>
                    <tr>
                        <td>Due Date:</td>
                        <td>{{.Invoice.DueDate.Format "02.01.2006"}}</td>
                    </tr>
                    <tr>
                        <td>Status:</td>
                        <td><span class="status-badge status-{{.Invoice.Status}}">{{.Invoice.Status}}</span></td>
                    </tr>
                </table>
            </div>
        </div>
    </div>

    <!-- Billing Information -->
    <div class="billing-section">
        <div class="bill-to">
            <h3>Bill To:</h3>
            {{if .Invoice.Customer}}
            <div class="address-box">
                <strong>{{.Invoice.Customer.GetDisplayName}}</strong><br>
                {{if .Invoice.Customer.Email}}{{.Invoice.Customer.Email}}<br>{{end}}
                {{if .Invoice.Customer.PhoneNumber}}{{.Invoice.Customer.PhoneNumber}}<br>{{end}}
                {{if .Invoice.Customer.Street}}{{.Invoice.Customer.Street}}{{if .Invoice.Customer.HouseNumber}} {{.Invoice.Customer.HouseNumber}}{{end}}<br>{{end}}
                {{if .Invoice.Customer.ZIP}}{{.Invoice.Customer.ZIP}} {{end}}{{if .Invoice.Customer.City}}{{.Invoice.Customer.City}}{{end}}
            </div>
            {{else}}
            <div class="address-box">Customer information not available</div>
            {{end}}
        </div>
        
        {{if .Invoice.Job}}
        <div class="job-info">
            <h3>Job Reference:</h3>
            <div class="address-box">
                <strong>{{.Invoice.Job.Description}}</strong><br>
                {{if .Invoice.Job.StartDate}}<small>Start: {{.Invoice.Job.StartDate.Format "02.01.2006"}}</small><br>{{end}}
                {{if .Invoice.Job.EndDate}}<small>End: {{.Invoice.Job.EndDate.Format "02.01.2006"}}</small>{{end}}
            </div>
        </div>
        {{end}}
    </div>

    <!-- Line Items -->
    <div class="line-items">
        <h3>Invoice Items</h3>
        <table class="items-table">
            <thead>
                <tr>
                    <th>Description</th>
                    <th width="10%">Quantity</th>
                    <th width="12%">Unit Price</th>
                    <th width="12%" class="text-right">Total</th>
                </tr>
            </thead>
            <tbody>
                {{if .Invoice.LineItems}}
                {{range .Invoice.LineItems}}
                <tr>
                    <td>
                        <strong>{{.Description}}</strong>
                        {{if eq .ItemType "device"}}
                            <br><small>Device Item</small>
                        {{else if eq .ItemType "package"}}
                            <br><small>Package Item</small>
                        {{else if eq .ItemType "service"}}
                            <br><small>Service</small>
                        {{end}}
                    </td>
                    <td>{{printf "%.2f" .Quantity}}</td>
                    <td>{{$.Settings.CurrencySymbol}}{{printf "%.2f" .UnitPrice}}</td>
                    <td class="text-right">{{$.Settings.CurrencySymbol}}{{printf "%.2f" .TotalPrice}}</td>
                </tr>
                {{end}}
                {{else}}
                <tr>
                    <td colspan="4" style="text-align: center; padding: 20px; color: #666;">
                        No line items have been added to this invoice.
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <!-- Totals and Notes -->
    <div class="totals-section">
        {{if .Invoice.Notes}}
        <div class="notes">
            <h3>Notes:</h3>
            <div class="address-box">{{.Invoice.Notes}}</div>
        </div>
        {{end}}
        
        <div class="totals">
            <table class="totals-table">
                <tr>
                    <td><strong>Subtotal:</strong></td>
                    <td class="text-right">{{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.Subtotal}}</td>
                </tr>
                <tr>
                    <td><strong>Tax ({{printf "%.1f" .Invoice.TaxRate}}%):</strong></td>
                    <td class="text-right">{{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.TaxAmount}}</td>
                </tr>
                {{if gt .Invoice.DiscountAmount 0}}
                <tr>
                    <td><strong>Discount:</strong></td>
                    <td class="text-right">-{{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.DiscountAmount}}</td>
                </tr>
                {{end}}
                <tr class="total-row">
                    <td><strong>Total Amount:</strong></td>
                    <td class="text-right"><strong>{{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.TotalAmount}}</strong></td>
                </tr>
            </table>
        </div>
    </div>

    <!-- Terms and Conditions -->
    {{if .Invoice.TermsConditions}}
    <div style="margin-bottom: 30px;">
        <h3>Terms & Conditions:</h3>
        <div class="address-box" style="font-size: 11px;">{{.Invoice.TermsConditions}}</div>
    </div>
    {{end}}

    <!-- Footer -->
    <div class="footer-info">
        {{if .Company.TaxNumber}}<strong>Tax Number:</strong> {{.Company.TaxNumber}} | {{end}}
        {{if .Company.VATNumber}}<strong>VAT Number:</strong> {{.Company.VATNumber}} | {{end}}
        {{if .Company.Email}}{{.Company.Email}} | {{end}}
        {{if .Company.Website}}{{.Company.Website}}{{end}}
        <br><br>
        <small>Generated on {{.GeneratedAt.Format "02.01.2006 15:04:05"}}</small>
    </div>
</body>
</html>`

	// Create template
	tmpl, err := template.New("invoice").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	// Prepare template data
	data := struct {
		Invoice     *models.Invoice
		Company     *models.CompanySettings
		Settings    *models.InvoiceSettings
		GeneratedAt time.Time
	}{
		Invoice:     invoice,
		Company:     company,
		Settings:    settings,
		GeneratedAt: time.Now(),
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return buf.String(), nil
}

// getDefaultCompanySettings returns default company settings
func (s *PDFServiceNew) getDefaultCompanySettings() *models.CompanySettings {
	companyName := "RentalCore Company"
	addressLine1 := "123 Business Street"
	city := "Business City"
	country := "Germany"
	phone := "+49 123 456 789"
	email := "info@rentalcore.com"

	return &models.CompanySettings{
		CompanyName:  companyName,
		AddressLine1: &addressLine1,
		City:         &city,
		Country:      &country,
		Phone:        &phone,
		Email:        &email,
	}
}

// getDefaultInvoiceSettings returns default invoice settings
func (s *PDFServiceNew) getDefaultInvoiceSettings() *models.InvoiceSettings {
	return &models.InvoiceSettings{
		InvoiceNumberPrefix:     "RE",
		InvoiceNumberFormat:     "{prefix}{sequence:4}",
		DefaultPaymentTerms:     30,
		DefaultTaxRate:          19.0,
		AutoCalculateRentalDays: true,
		ShowLogoOnInvoice:       true,
		CurrencySymbol:          "€",
		CurrencyCode:            "EUR",
		DateFormat:              "DD.MM.YYYY",
	}
}