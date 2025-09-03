package services

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"
)

type EmailService struct {
	config *config.EmailConfig
}

func NewEmailService(emailConfig *config.EmailConfig) *EmailService {
	return &EmailService{
		config: emailConfig,
	}
}

func NewEmailServiceFromCompany(company *models.CompanySettings) *EmailService {
	// Create config from company settings
	emailConfig := &config.EmailConfig{
		SMTPHost:     getStringValue(company.SMTPHost),
		SMTPPort:     getIntValue(company.SMTPPort, 587),
		SMTPUsername: getStringValue(company.SMTPUsername),
		SMTPPassword: getStringValue(company.SMTPPassword),
		FromEmail:    getStringValue(company.SMTPFromEmail),
		FromName:     getStringValue(company.SMTPFromName),
		UseTLS:       getBoolValue(company.SMTPUseTLS, true),
	}
	
	return &EmailService{
		config: emailConfig,
	}
}

// EmailData represents data for email templates
type EmailData struct {
	Invoice      *models.Invoice
	Company      *models.CompanySettings
	Customer     *models.Customer
	Settings     *models.InvoiceSettings
	InvoiceURL   string
	PaymentURL   string
	SupportEmail string
}

// SendInvoiceEmail sends an invoice via email
func (s *EmailService) SendInvoiceEmail(emailData *EmailData, pdfAttachment []byte) error {
	if emailData.Customer == nil || emailData.Customer.Email == nil || *emailData.Customer.Email == "" {
		return fmt.Errorf("customer email not available")
	}

	// Generate email content
	subject, err := s.generateEmailSubject(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate email subject: %v", err)
	}

	htmlBody, err := s.generateEmailHTML(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate email HTML: %v", err)
	}

	textBody, err := s.generateEmailText(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate email text: %v", err)
	}

	// Send email
	return s.sendEmail(
		[]string{*emailData.Customer.Email},
		subject,
		textBody,
		htmlBody,
		pdfAttachment,
		fmt.Sprintf("Invoice_%s.pdf", emailData.Invoice.InvoiceNumber),
	)
}

// SendTestEmail sends a test email
func (s *EmailService) SendTestEmail(toEmail string, testData *EmailData) error {
	subject := "Test Email from RentalCore Invoice System"
	
	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Test Email</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #007bff;">🧪 Test Email - RentalCore Invoice System</h2>
        
        <p>This is a test email from your RentalCore invoice system.</p>
        
        <div style="background-color: #f8f9fa; border-left: 4px solid #007bff; padding: 15px; margin: 20px 0;">
            <h3>Email Configuration Status</h3>
            <ul>
                <li><strong>SMTP Host:</strong> ` + s.config.SMTPHost + `</li>
                <li><strong>SMTP Port:</strong> ` + strconv.Itoa(s.config.SMTPPort) + `</li>
                <li><strong>From Email:</strong> ` + s.config.FromEmail + `</li>
                <li><strong>Sent At:</strong> ` + time.Now().Format("2006-01-02 15:04:05") + `</li>
            </ul>
        </div>
        
        <p>If you received this email, your email configuration is working correctly!</p>
        
        <hr style="border: none; border-top: 1px solid #ddd; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            This email was sent from RentalCore - The core of your rental business<br>
            <a href="mailto:` + s.config.FromEmail + `">` + s.config.FromEmail + `</a>
        </p>
    </div>
</body>
</html>
`

	textBody := `
Test Email - RentalCore Invoice System

This is a test email from your RentalCore invoice system.

Email Configuration Status:
- SMTP Host: ` + s.config.SMTPHost + `
- SMTP Port: ` + strconv.Itoa(s.config.SMTPPort) + `
- From Email: ` + s.config.FromEmail + `
- Sent At: ` + time.Now().Format("2006-01-02 15:04:05") + `

If you received this email, your email configuration is working correctly!

---
RentalCore - The core of your rental business
` + s.config.FromEmail

	return s.sendEmail([]string{toEmail}, subject, textBody, htmlBody, nil, "")
}

// generateEmailSubject creates the email subject line
func (s *EmailService) generateEmailSubject(data *EmailData) (string, error) {
	// Default template
	subjectTemplate := "Invoice {{.Invoice.InvoiceNumber}} from {{.Company.CompanyName}}"
	
	// Try to use custom template if available
	// This would typically come from invoice settings
	
	tmpl, err := template.New("subject").Parse(subjectTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// generateEmailHTML creates the HTML email body
func (s *EmailService) generateEmailHTML(data *EmailData) (string, error) {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Invoice {{.Invoice.InvoiceNumber}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #007bff; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .invoice-details { background-color: #f8f9fa; padding: 15px; border-left: 4px solid #007bff; }
        .amount-due { font-size: 24px; color: #007bff; font-weight: bold; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        .button { display: inline-block; background-color: #007bff; color: white; padding: 12px 24px; 
                 text-decoration: none; border-radius: 4px; margin: 10px 0; }
        .warning { background-color: #dc3545; color: white; padding: 10px; text-align: center; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        {{if .Invoice.IsOverdue}}
        <div class="warning">
            <strong>⚠️ OVERDUE NOTICE</strong> - This invoice is past due and requires immediate attention.
        </div>
        {{end}}
        
        <div class="header">
            <h1>{{.Company.CompanyName}}</h1>
            <p>Invoice {{.Invoice.InvoiceNumber}}</p>
        </div>
        
        <div class="content">
            <p>Dear {{.Customer.GetDisplayName}},</p>
            
            <p>Please find attached your invoice for the services provided. Below are the details:</p>
            
            <div class="invoice-details">
                <h3>Invoice Details</h3>
                <table style="width: 100%; border-collapse: collapse;">
                    <tr>
                        <td><strong>Invoice Number:</strong></td>
                        <td>{{.Invoice.InvoiceNumber}}</td>
                    </tr>
                    <tr>
                        <td><strong>Issue Date:</strong></td>
                        <td>{{.Invoice.IssueDate.Format "January 2, 2006"}}</td>
                    </tr>
                    <tr>
                        <td><strong>Due Date:</strong></td>
                        <td>{{.Invoice.DueDate.Format "January 2, 2006"}}</td>
                    </tr>
                    {{if .Invoice.Job}}
                    <tr>
                        <td><strong>Job Reference:</strong></td>
                        <td>{{.Invoice.Job.Description}}</td>
                    </tr>
                    {{end}}
                </table>
            </div>
            
            <div style="text-align: center; margin: 30px 0;">
                <div class="amount-due">
                    Total Amount: {{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.TotalAmount}}
                </div>
                {{if gt .Invoice.BalanceDue 0}}
                <div style="font-size: 18px; color: #dc3545; margin-top: 10px;">
                    <strong>Balance Due: {{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.BalanceDue}}</strong>
                </div>
                {{end}}
            </div>
            
            {{if .InvoiceURL}}
            <div style="text-align: center;">
                <a href="{{.InvoiceURL}}" class="button">View Invoice Online</a>
            </div>
            {{end}}
            
            {{if .Invoice.PaymentTerms}}
            <div style="margin: 20px 0;">
                <h4>Payment Terms:</h4>
                <p>{{.Invoice.PaymentTerms}}</p>
            </div>
            {{end}}
            
            {{if .Invoice.Notes}}
            <div style="margin: 20px 0;">
                <h4>Additional Notes:</h4>
                <p>{{.Invoice.Notes}}</p>
            </div>
            {{end}}
            
            <p>If you have any questions about this invoice, please don't hesitate to contact us.</p>
            
            <p>Thank you for your business!</p>
            
            <p>Best regards,<br>
            {{.Company.CompanyName}}<br>
            {{if .Company.Phone}}Phone: {{.Company.Phone}}<br>{{end}}
            {{if .Company.Email}}Email: {{.Company.Email}}<br>{{end}}
            {{if .Company.Website}}Website: {{.Company.Website}}{{end}}
            </p>
        </div>
        
        <div class="footer">
            <p>This email was sent from {{.Company.CompanyName}}<br>
            {{if .Company.AddressLine1}}{{.Company.AddressLine1}}<br>{{end}}
            {{if .Company.City}}{{.Company.City}}{{if .Company.PostalCode}}, {{.Company.PostalCode}}{{end}}<br>{{end}}
            {{if .Company.Country}}{{.Company.Country}}{{end}}
            </p>
            
            {{if or .Company.TaxNumber .Company.VATNumber}}
            <p style="font-size: 11px;">
                {{if .Company.TaxNumber}}Tax Number: {{.Company.TaxNumber}} | {{end}}
                {{if .Company.VATNumber}}VAT Number: {{.Company.VATNumber}}{{end}}
            </p>
            {{end}}
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateEmailText creates the plain text email body
func (s *EmailService) generateEmailText(data *EmailData) (string, error) {
	textTemplate := `
{{if .Invoice.IsOverdue}}⚠️  OVERDUE NOTICE - This invoice is past due and requires immediate attention.

{{end}}INVOICE {{.Invoice.InvoiceNumber}}
{{.Company.CompanyName}}

Dear {{.Customer.GetDisplayName}},

Please find attached your invoice for the services provided. Below are the details:

Invoice Details:
- Invoice Number: {{.Invoice.InvoiceNumber}}
- Issue Date: {{.Invoice.IssueDate.Format "January 2, 2006"}}
- Due Date: {{.Invoice.DueDate.Format "January 2, 2006"}}
{{if .Invoice.Job}}- Job Reference: {{.Invoice.Job.Description}}
{{end}}
Total Amount: {{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.TotalAmount}}
{{if gt .Invoice.BalanceDue 0}}Balance Due: {{.Settings.CurrencySymbol}}{{printf "%.2f" .Invoice.BalanceDue}}

{{end}}{{if .InvoiceURL}}View Invoice Online: {{.InvoiceURL}}

{{end}}{{if .Invoice.PaymentTerms}}Payment Terms:
{{.Invoice.PaymentTerms}}

{{end}}{{if .Invoice.Notes}}Additional Notes:
{{.Invoice.Notes}}

{{end}}If you have any questions about this invoice, please don't hesitate to contact us.

Thank you for your business!

Best regards,
{{.Company.CompanyName}}
{{if .Company.Phone}}Phone: {{.Company.Phone}}
{{end}}{{if .Company.Email}}Email: {{.Company.Email}}
{{end}}{{if .Company.Website}}Website: {{.Company.Website}}
{{end}}
---
{{if .Company.AddressLine1}}{{.Company.AddressLine1}}
{{end}}{{if .Company.City}}{{.Company.City}}{{if .Company.PostalCode}}, {{.Company.PostalCode}}{{end}}
{{end}}{{if .Company.Country}}{{.Company.Country}}
{{end}}
{{if or .Company.TaxNumber .Company.VATNumber}}
{{if .Company.TaxNumber}}Tax Number: {{.Company.TaxNumber}}{{end}}{{if and .Company.TaxNumber .Company.VATNumber}} | {{end}}{{if .Company.VATNumber}}VAT Number: {{.Company.VATNumber}}{{end}}
{{end}}
`

	tmpl, err := template.New("email_text").Parse(textTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// sendEmail sends an email with optional PDF attachment
func (s *EmailService) sendEmail(to []string, subject, textBody, htmlBody string, attachment []byte, attachmentName string) error {
	// Check if SMTP is configured
	if s.config.SMTPHost == "localhost" && s.config.SMTPUsername == "" {
		return fmt.Errorf("SMTP not configured - please set email configuration in config.json: smtp_host, smtp_port, smtp_username, smtp_password")
	}

	// Create SMTP connection
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	
	// Setup authentication
	var auth smtp.Auth
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// Try without TLS
		return s.sendEmailPlain(to, subject, textBody, htmlBody, attachment, attachmentName)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", recipient, err)
		}
	}

	// Create message
	message := s.createMIMEMessage(to, subject, textBody, htmlBody, attachment, attachmentName)

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %v", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}

// sendEmailPlain sends email without TLS (fallback)
func (s *EmailService) sendEmailPlain(to []string, subject, textBody, htmlBody string, attachment []byte, attachmentName string) error {
	message := s.createMIMEMessage(to, subject, textBody, htmlBody, attachment, attachmentName)
	
	var auth smtp.Auth
	if s.config.SMTPUsername != "" && s.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	}

	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	return smtp.SendMail(addr, auth, s.config.FromEmail, to, []byte(message))
}

// createMIMEMessage creates a MIME message with optional attachment
func (s *EmailService) createMIMEMessage(to []string, subject, textBody, htmlBody string, attachment []byte, attachmentName string) string {
	boundary := "boundary-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	
	var message strings.Builder
	
	// Headers
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	
	if attachment != nil {
		message.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	} else {
		message.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	}
	message.WriteString("\r\n")
	
	// Text part
	message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	message.WriteString(textBody)
	message.WriteString("\r\n\r\n")
	
	// HTML part
	message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	message.WriteString(htmlBody)
	message.WriteString("\r\n\r\n")
	
	// Attachment
	if attachment != nil && attachmentName != "" {
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString(fmt.Sprintf("Content-Type: application/pdf; name=\"%s\"\r\n", attachmentName))
		message.WriteString("Content-Transfer-Encoding: base64\r\n")
		message.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", attachmentName))
		
		// Encode attachment as base64
		encoded := s.encodeBase64(attachment)
		message.WriteString(encoded)
		message.WriteString("\r\n")
	}
	
	message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	
	return message.String()
}

// encodeBase64 encodes data as base64 with line breaks
func (s *EmailService) encodeBase64(data []byte) string {
	// Simple base64 encoding with line breaks every 76 characters
	encoded := ""
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}
		
		encoded += string(base64Table[b1>>2])
		encoded += string(base64Table[((b1&0x03)<<4)|(b2>>4)])
		if i+1 < len(data) {
			encoded += string(base64Table[((b2&0x0f)<<2)|(b3>>6)])
		} else {
			encoded += "="
		}
		if i+2 < len(data) {
			encoded += string(base64Table[b3&0x3f])
		} else {
			encoded += "="
		}
		
		if len(encoded)%76 == 0 {
			encoded += "\r\n"
		}
	}
	
	return encoded
}

// Helper functions for safe pointer access
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getIntValue(ptr *int, defaultValue int) int {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

func getBoolValue(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

