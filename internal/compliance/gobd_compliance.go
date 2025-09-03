package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

// GoBDCompliance handles German GoBD (Grundsätze zur ordnungsmäßigen Führung und Aufbewahrung von Büchern) compliance
type GoBDCompliance struct {
	db           *gorm.DB
	archivePath  string
	auditLogger  *AuditLogger
	retentionMgr *RetentionManager
}

// GoBDRecord represents a GoBD-compliant archived record
type GoBDRecord struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	DocumentType    string    `json:"document_type" gorm:"not null;index"` // invoice, receipt, contract, etc.
	DocumentID      string    `json:"document_id" gorm:"not null;index"`   // Original document ID
	OriginalData    string    `json:"original_data" gorm:"type:longtext"`  // Original JSON data
	DataHash        string    `json:"data_hash" gorm:"not null;index"`     // SHA256 hash for integrity
	ArchiveDate     time.Time `json:"archive_date" gorm:"not null;index"`
	RetentionDate   time.Time `json:"retention_date" gorm:"not null;index"` // When it can be deleted
	DigitalSign     string    `json:"digital_sign" gorm:"type:text"`        // Digital signature
	UserID          uint      `json:"user_id" gorm:"index"`                 // User who created the record
	CompanyID       uint      `json:"company_id" gorm:"index"`              // Company context
	IsImmutable     bool      `json:"is_immutable" gorm:"default:true"`     // GoBD requires immutability
	ArchiveFileName string    `json:"archive_file_name"`                    // Physical file location
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (GoBDRecord) TableName() string {
	return "gobd_records"
}

// AuditEvent represents a GoBD-compliant audit log entry
type AuditEvent struct {
	ID            uint                   `json:"id" gorm:"primaryKey"`
	EventType     string                 `json:"event_type" gorm:"not null;index"`     // CREATE, READ, UPDATE, DELETE, ARCHIVE
	ObjectType    string                 `json:"object_type" gorm:"not null;index"`    // invoice, customer, device, etc.
	ObjectID      string                 `json:"object_id" gorm:"not null;index"`      // ID of the affected object
	UserID        uint                   `json:"user_id" gorm:"not null;index"`        // User performing the action
	Username      string                 `json:"username" gorm:"not null"`             // Username for accountability
	Action        string                 `json:"action" gorm:"not null"`               // Detailed action description
	OldValues     string                 `json:"old_values" gorm:"type:text"`          // JSON of old values (for updates)
	NewValues     string                 `json:"new_values" gorm:"type:text"`          // JSON of new values
	IPAddress     string                 `json:"ip_address" gorm:"not null"`           // Client IP for tracking
	UserAgent     string                 `json:"user_agent" gorm:"type:text"`          // Browser/client info
	SessionID     string                 `json:"session_id" gorm:"index"`              // Session tracking
	Context       map[string]interface{} `json:"context" gorm:"type:json"`             // Additional context
	EventHash     string                 `json:"event_hash" gorm:"not null;unique"`    // Hash for immutability
	PreviousHash  string                 `json:"previous_hash" gorm:"index"`           // Chain for integrity
	IsCompliant   bool                   `json:"is_compliant" gorm:"default:true"`     // GoBD compliance flag
	RetentionDate time.Time              `json:"retention_date" gorm:"not null;index"` // When it can be deleted
	Timestamp     time.Time              `json:"timestamp" gorm:"not null;index"`      // Event timestamp
	CreatedAt     time.Time              `json:"created_at" gorm:"autoCreateTime"`
}

func (AuditEvent) TableName() string {
	return "audit_events"
}

// RetentionPolicy defines data retention rules according to German law
type RetentionPolicy struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	DocumentType     string    `json:"document_type" gorm:"not null;unique"` // invoice, receipt, contract, etc.
	RetentionYears   int       `json:"retention_years" gorm:"not null"`      // Legal retention period
	LegalBasis       string    `json:"legal_basis" gorm:"not null"`          // Reference to law (e.g., "§ 147 AO")
	Description      string    `json:"description" gorm:"type:text"`         // Human-readable description
	IsActive         bool      `json:"is_active" gorm:"default:true"`
	AutoDeleteAfter  bool      `json:"auto_delete_after" gorm:"default:false"` // Auto-delete after retention
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (RetentionPolicy) TableName() string {
	return "retention_policies"
}

// NewGoBDCompliance creates a new GoBD compliance manager
func NewGoBDCompliance(db *gorm.DB, archivePath string) (*GoBDCompliance, error) {
	// Ensure archive directory exists
	if err := os.MkdirAll(archivePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	gbc := &GoBDCompliance{
		db:          db,
		archivePath: archivePath,
	}

	// Initialize audit logger
	auditLogger, err := NewAuditLogger(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}
	gbc.auditLogger = auditLogger

	// Initialize retention manager
	retentionMgr, err := NewRetentionManager(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize retention manager: %w", err)
	}
	gbc.retentionMgr = retentionMgr

	// Auto-migrate tables
	if err := gbc.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate GoBD tables: %w", err)
	}

	// Initialize default retention policies
	if err := gbc.initializeDefaultPolicies(); err != nil {
		log.Printf("Warning: Failed to initialize default retention policies: %v", err)
	}

	return gbc, nil
}

// migrate auto-migrates GoBD compliance tables
func (gbc *GoBDCompliance) migrate() error {
	// Migration disabled - tables should be created manually
	log.Printf("GoBD compliance table migration disabled")
	return nil
}

// ArchiveInvoice archives an invoice in GoBD-compliant format
func (gbc *GoBDCompliance) ArchiveInvoice(invoice *models.Invoice, userID uint) error {
	// Serialize invoice data
	originalData, err := json.Marshal(invoice)
	if err != nil {
		return fmt.Errorf("failed to serialize invoice: %w", err)
	}

	// Calculate data hash for integrity
	dataHash := gbc.calculateHash(originalData)

	// Get retention policy for invoices
	retentionDate, err := gbc.retentionMgr.GetRetentionDate("invoice")
	if err != nil {
		return fmt.Errorf("failed to get retention date: %w", err)
	}

	// Create archive file
	fileName := fmt.Sprintf("invoice_%s_%s.json", invoice.InvoiceNumber, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(gbc.archivePath, "invoices", fileName)
	
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	if err := os.WriteFile(filePath, originalData, 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	// Create GoBD record
	record := &GoBDRecord{
		DocumentType:    "invoice",
		DocumentID:      fmt.Sprintf("%d", invoice.InvoiceID),
		OriginalData:    string(originalData),
		DataHash:        dataHash,
		ArchiveDate:     time.Now(),
		RetentionDate:   retentionDate,
		UserID:          userID,
		CompanyID:       1, // TODO: Get from context
		IsImmutable:     true,
		ArchiveFileName: fileName,
	}

	// Generate digital signature
	record.DigitalSign = gbc.generateDigitalSignature(record)

	// Save to database
	if err := gbc.db.Create(record).Error; err != nil {
		// Clean up file if database save fails
		os.Remove(filePath)
		return fmt.Errorf("failed to create GoBD record: %w", err)
	}

	// Log audit event
	return gbc.auditLogger.LogEvent("ARCHIVE", "invoice", fmt.Sprintf("%d", invoice.InvoiceID), userID, "Invoice archived for GoBD compliance", nil, map[string]interface{}{
		"invoice_number": invoice.InvoiceNumber,
		"archive_file":   fileName,
		"data_hash":      dataHash,
	})
}

// ArchiveDocument archives any document in GoBD-compliant format
func (gbc *GoBDCompliance) ArchiveDocument(documentType, documentID string, data interface{}, userID uint) error {
	// Serialize document data
	originalData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// Calculate data hash for integrity
	dataHash := gbc.calculateHash(originalData)

	// Get retention policy
	retentionDate, err := gbc.retentionMgr.GetRetentionDate(documentType)
	if err != nil {
		// Use default 10 years if no specific policy
		retentionDate = time.Now().AddDate(10, 0, 0)
	}

	// Create archive file
	fileName := fmt.Sprintf("%s_%s_%s.json", documentType, documentID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(gbc.archivePath, documentType, fileName)
	
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	if err := os.WriteFile(filePath, originalData, 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	// Create GoBD record
	record := &GoBDRecord{
		DocumentType:    documentType,
		DocumentID:      documentID,
		OriginalData:    string(originalData),
		DataHash:        dataHash,
		ArchiveDate:     time.Now(),
		RetentionDate:   retentionDate,
		UserID:          userID,
		CompanyID:       1, // TODO: Get from context
		IsImmutable:     true,
		ArchiveFileName: fileName,
	}

	// Generate digital signature
	record.DigitalSign = gbc.generateDigitalSignature(record)

	// Save to database
	if err := gbc.db.Create(record).Error; err != nil {
		// Clean up file if database save fails
		os.Remove(filePath)
		return fmt.Errorf("failed to create GoBD record: %w", err)
	}

	// Log audit event
	return gbc.auditLogger.LogEvent("ARCHIVE", documentType, documentID, userID, "Document archived for GoBD compliance", nil, map[string]interface{}{
		"document_type": documentType,
		"archive_file":  fileName,
		"data_hash":     dataHash,
	})
}

// VerifyIntegrity verifies the integrity of an archived document
func (gbc *GoBDCompliance) VerifyIntegrity(recordID uint) (bool, error) {
	var record GoBDRecord
	if err := gbc.db.First(&record, recordID).Error; err != nil {
		return false, fmt.Errorf("failed to find GoBD record: %w", err)
	}

	// Verify data hash
	currentHash := gbc.calculateHash([]byte(record.OriginalData))
	if currentHash != record.DataHash {
		return false, fmt.Errorf("data integrity check failed: hash mismatch")
	}

	// Verify file exists and matches
	filePath := filepath.Join(gbc.archivePath, record.DocumentType, record.ArchiveFileName)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read archive file: %w", err)
	}

	fileHash := gbc.calculateHash(fileData)
	if fileHash != record.DataHash {
		return false, fmt.Errorf("file integrity check failed: hash mismatch")
	}

	return true, nil
}

// GetArchivedDocument retrieves an archived document
func (gbc *GoBDCompliance) GetArchivedDocument(documentType, documentID string) (*GoBDRecord, error) {
	var record GoBDRecord
	if err := gbc.db.Where("document_type = ? AND document_id = ?", documentType, documentID).First(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to find archived document: %w", err)
	}

	// Verify integrity before returning
	if valid, err := gbc.VerifyIntegrity(record.ID); err != nil || !valid {
		return nil, fmt.Errorf("archived document failed integrity check: %v", err)
	}

	return &record, nil
}

// CleanupExpiredRecords removes records that have passed their retention period
func (gbc *GoBDCompliance) CleanupExpiredRecords() error {
	now := time.Now()
	
	// Find expired records
	var expiredRecords []GoBDRecord
	if err := gbc.db.Where("retention_date < ?", now).Find(&expiredRecords).Error; err != nil {
		return fmt.Errorf("failed to find expired records: %w", err)
	}

	for _, record := range expiredRecords {
		// Check if auto-deletion is allowed for this document type
		canDelete, err := gbc.retentionMgr.CanAutoDelete(record.DocumentType)
		if err != nil {
			log.Printf("Warning: Failed to check auto-delete policy for %s: %v", record.DocumentType, err)
			continue
		}

		if !canDelete {
			log.Printf("Auto-deletion disabled for document type %s, skipping record %d", record.DocumentType, record.ID)
			continue
		}

		// Delete archive file
		filePath := filepath.Join(gbc.archivePath, record.DocumentType, record.ArchiveFileName)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Warning: Failed to delete archive file %s: %v", filePath, err)
		}

		// Delete database record
		if err := gbc.db.Delete(&record).Error; err != nil {
			log.Printf("Warning: Failed to delete GoBD record %d: %v", record.ID, err)
		} else {
			log.Printf("Deleted expired GoBD record %d (%s)", record.ID, record.DocumentType)
		}
	}

	return nil
}

// GetComplianceReport generates a compliance report
func (gbc *GoBDCompliance) GetComplianceReport() (*ComplianceReport, error) {
	report := &ComplianceReport{
		GeneratedAt: time.Now(),
	}

	// Count records by type
	var results []struct {
		DocumentType string
		Count        int64
	}
	
	if err := gbc.db.Model(&GoBDRecord{}).
		Select("document_type, COUNT(*) as count").
		Group("document_type").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get document counts: %w", err)
	}

	report.ArchivedDocuments = make(map[string]int64)
	for _, result := range results {
		report.ArchivedDocuments[result.DocumentType] = result.Count
	}

	// Get total archive size
	if err := filepath.Walk(gbc.archivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			report.TotalArchiveSize += info.Size()
		}
		return nil
	}); err != nil {
		log.Printf("Warning: Failed to calculate archive size: %v", err)
	}

	// Count upcoming expirations
	nextMonth := time.Now().AddDate(0, 1, 0)
	if err := gbc.db.Model(&GoBDRecord{}).
		Where("retention_date BETWEEN ? AND ?", time.Now(), nextMonth).
		Count(&report.ExpiringRecords).Error; err != nil {
		log.Printf("Warning: Failed to count expiring records: %v", err)
	}

	// Get audit statistics
	auditStats, err := gbc.auditLogger.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get audit statistics: %v", err)
	} else {
		report.AuditEvents = auditStats.TotalEvents
		report.IntegrityChecks = auditStats.IntegrityChecks
	}

	return report, nil
}

// calculateHash calculates SHA256 hash of data
func (gbc *GoBDCompliance) calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// generateDigitalSignature generates a simple digital signature for the record
func (gbc *GoBDCompliance) generateDigitalSignature(record *GoBDRecord) string {
	// For production, this should use proper cryptographic signing
	// For now, using a hash-based approach
	signatureData := fmt.Sprintf("%s:%s:%s:%s",
		record.DocumentType,
		record.DocumentID,
		record.DataHash,
		record.ArchiveDate.Format(time.RFC3339),
	)
	return gbc.calculateHash([]byte(signatureData))
}

// initializeDefaultPolicies creates default retention policies according to German law
func (gbc *GoBDCompliance) initializeDefaultPolicies() error {
	policies := []RetentionPolicy{
		{
			DocumentType:    "invoice",
			RetentionYears:  10,
			LegalBasis:      "§ 147 AO (Abgabenordnung)",
			Description:     "Rechnungen müssen 10 Jahre aufbewahrt werden",
			IsActive:        true,
			AutoDeleteAfter: false,
		},
		{
			DocumentType:    "receipt",
			RetentionYears:  10,
			LegalBasis:      "§ 147 AO (Abgabenordnung)",
			Description:     "Belege müssen 10 Jahre aufbewahrt werden",
			IsActive:        true,
			AutoDeleteAfter: false,
		},
		{
			DocumentType:    "contract",
			RetentionYears:  10,
			LegalBasis:      "§ 147 AO (Abgabenordnung)",
			Description:     "Verträge müssen 10 Jahre aufbewahrt werden",
			IsActive:        true,
			AutoDeleteAfter: false,
		},
		{
			DocumentType:    "customer_data",
			RetentionYears:  6,
			LegalBasis:      "§ 257 HGB (Handelsgesetzbuch)",
			Description:     "Kundendaten müssen 6 Jahre aufbewahrt werden",
			IsActive:        true,
			AutoDeleteAfter: true, // GDPR compliance
		},
		{
			DocumentType:    "audit_log",
			RetentionYears:  10,
			LegalBasis:      "GoBD (Grundsätze ordnungsmäßiger Buchführung)",
			Description:     "Audit-Logs müssen 10 Jahre aufbewahrt werden",
			IsActive:        true,
			AutoDeleteAfter: false,
		},
	}

	for _, policy := range policies {
		var existing RetentionPolicy
		err := gbc.db.Where("document_type = ?", policy.DocumentType).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := gbc.db.Create(&policy).Error; err != nil {
				return fmt.Errorf("failed to create retention policy for %s: %w", policy.DocumentType, err)
			}
		}
	}

	return nil
}

// ComplianceReport represents a GoBD compliance report
type ComplianceReport struct {
	GeneratedAt        time.Time         `json:"generated_at"`
	ArchivedDocuments  map[string]int64  `json:"archived_documents"`
	TotalArchiveSize   int64             `json:"total_archive_size"`
	ExpiringRecords    int64             `json:"expiring_records"`
	AuditEvents        int64             `json:"audit_events"`
	IntegrityChecks    int64             `json:"integrity_checks"`
	ComplianceStatus   string            `json:"compliance_status"`
	Recommendations    []string          `json:"recommendations"`
}