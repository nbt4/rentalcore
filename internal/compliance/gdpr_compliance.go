package compliance

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gorm.io/gorm"
)

// GDPRDataType represents different types of personal data
type GDPRDataType string

const (
	PersonalIdentity GDPRDataType = "personal_identity"
	ContactInfo      GDPRDataType = "contact_info"
	FinancialData    GDPRDataType = "financial_data"
	BehavioralData   GDPRDataType = "behavioral_data"
	TechnicalData    GDPRDataType = "technical_data"
)

// ConsentRecord tracks user consent for data processing
type ConsentRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null;index"`
	DataType     string    `json:"data_type" gorm:"not null"`
	Purpose      string    `json:"purpose" gorm:"not null"`
	ConsentGiven bool      `json:"consent_given" gorm:"not null"`
	ConsentDate  time.Time `json:"consent_date" gorm:"not null"`
	ExpiryDate   *time.Time `json:"expiry_date"`
	LegalBasis   string    `json:"legal_basis" gorm:"not null"` // Art. 6 GDPR basis
	WithdrawnAt  *time.Time `json:"withdrawn_at"`
	Version      string    `json:"version" gorm:"not null"` // Consent version
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DataProcessingRecord tracks all data processing activities
type DataProcessingRecord struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"not null;index"`
	DataType        string    `json:"data_type" gorm:"not null"`
	ProcessingType  string    `json:"processing_type" gorm:"not null"` // collection, storage, transfer, deletion
	Purpose         string    `json:"purpose" gorm:"not null"`
	LegalBasis      string    `json:"legal_basis" gorm:"not null"`
	DataController  string    `json:"data_controller" gorm:"not null"`
	DataProcessor   *string   `json:"data_processor"`
	Recipients      string    `json:"recipients"` // JSON array of recipients
	TransferCountry *string   `json:"transfer_country"`
	RetentionPeriod string    `json:"retention_period" gorm:"not null"`
	ProcessedAt     time.Time `json:"processed_at" gorm:"not null"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// DataSubjectRequest tracks GDPR data subject requests (Art. 15-22)
type DataSubjectRequest struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"user_id" gorm:"not null;index"`
	RequestType   string    `json:"request_type" gorm:"not null"` // access, rectification, erasure, portability, restriction, objection
	Status        string    `json:"status" gorm:"not null"`       // pending, processing, completed, rejected
	Description   string    `json:"description"`
	RequestedAt   time.Time `json:"requested_at" gorm:"not null"`
	ProcessedAt   *time.Time `json:"processed_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	ProcessorID   *uint     `json:"processor_id"` // User who processed the request
	Response      string    `json:"response"`      // Response to the request
	ResponseData  string    `json:"response_data"` // Exported data for portability requests
	Verification  string    `json:"verification"`  // Identity verification details
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EncryptedPersonalData stores encrypted personal data
type EncryptedPersonalData struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null;index"`
	DataType     string    `json:"data_type" gorm:"not null"`
	EncryptedData string   `json:"encrypted_data" gorm:"type:text;not null"`
	KeyVersion   string    `json:"key_version" gorm:"not null"`
	Algorithm    string    `json:"algorithm" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GDPRCompliance handles GDPR compliance operations
type GDPRCompliance struct {
	db           *gorm.DB
	encryptionKey []byte
	keyVersion   string
}

// NewGDPRCompliance creates a new GDPR compliance handler
func NewGDPRCompliance(db *gorm.DB, encryptionKey string) *GDPRCompliance {
	key := sha256.Sum256([]byte(encryptionKey))
	return &GDPRCompliance{
		db:           db,
		encryptionKey: key[:],
		keyVersion:   "v1.0",
	}
}

// RecordConsent records user consent for data processing
func (g *GDPRCompliance) RecordConsent(userID uint, dataType GDPRDataType, purpose, legalBasis, ipAddress, userAgent string, expiryDate *time.Time) error {
	consent := &ConsentRecord{
		UserID:       userID,
		DataType:     string(dataType),
		Purpose:      purpose,
		ConsentGiven: true,
		ConsentDate:  time.Now(),
		ExpiryDate:   expiryDate,
		LegalBasis:   legalBasis,
		Version:      "1.0",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	return g.db.Create(consent).Error
}

// WithdrawConsent withdraws user consent
func (g *GDPRCompliance) WithdrawConsent(userID uint, dataType GDPRDataType, purpose string) error {
	now := time.Now()
	return g.db.Model(&ConsentRecord{}).
		Where("user_id = ? AND data_type = ? AND purpose = ? AND consent_given = true AND withdrawn_at IS NULL", 
			userID, string(dataType), purpose).
		Update("withdrawn_at", now).Error
}

// CheckConsent checks if valid consent exists
func (g *GDPRCompliance) CheckConsent(userID uint, dataType GDPRDataType, purpose string) (bool, error) {
	var count int64
	err := g.db.Model(&ConsentRecord{}).
		Where("user_id = ? AND data_type = ? AND purpose = ? AND consent_given = true AND withdrawn_at IS NULL", 
			userID, string(dataType), purpose).
		Where("(expiry_date IS NULL OR expiry_date > ?)", time.Now()).
		Count(&count).Error
	
	return count > 0, err
}

// RecordDataProcessing records data processing activity
func (g *GDPRCompliance) RecordDataProcessing(userID uint, dataType GDPRDataType, processingType, purpose, legalBasis, controller string, processor *string, recipients []string, transferCountry *string, retentionPeriod string) error {
	recipientsJSON, _ := json.Marshal(recipients)
	
	var expiresAt *time.Time
	if retentionPeriod != "indefinite" {
		// Parse retention period and calculate expiry
		// This is a simplified example - you'd want more sophisticated parsing
		switch retentionPeriod {
		case "1_year":
			exp := time.Now().AddDate(1, 0, 0)
			expiresAt = &exp
		case "3_years":
			exp := time.Now().AddDate(3, 0, 0)
			expiresAt = &exp
		case "10_years":
			exp := time.Now().AddDate(10, 0, 0)
			expiresAt = &exp
		}
	}

	record := &DataProcessingRecord{
		UserID:          userID,
		DataType:        string(dataType),
		ProcessingType:  processingType,
		Purpose:         purpose,
		LegalBasis:      legalBasis,
		DataController:  controller,
		DataProcessor:   processor,
		Recipients:      string(recipientsJSON),
		TransferCountry: transferCountry,
		RetentionPeriod: retentionPeriod,
		ProcessedAt:     time.Now(),
		ExpiresAt:       expiresAt,
	}

	return g.db.Create(record).Error
}

// CreateDataSubjectRequest creates a new data subject request
func (g *GDPRCompliance) CreateDataSubjectRequest(userID uint, requestType, description string) error {
	request := &DataSubjectRequest{
		UserID:      userID,
		RequestType: requestType,
		Status:      "pending",
		Description: description,
		RequestedAt: time.Now(),
	}

	return g.db.Create(request).Error
}

// ProcessDataSubjectRequest processes a data subject request
func (g *GDPRCompliance) ProcessDataSubjectRequest(requestID uint, processorID uint, response string) error {
	now := time.Now()
	return g.db.Model(&DataSubjectRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":       "processing",
			"processor_id": processorID,
			"response":     response,
			"processed_at": now,
		}).Error
}

// CompleteDataSubjectRequest completes a data subject request
func (g *GDPRCompliance) CompleteDataSubjectRequest(requestID uint, responseData string) error {
	now := time.Now()
	return g.db.Model(&DataSubjectRequest{}).
		Where("id = ?", requestID).
		Updates(map[string]interface{}{
			"status":        "completed",
			"response_data": responseData,
			"completed_at":  now,
		}).Error
}

// EncryptPersonalData encrypts and stores personal data
func (g *GDPRCompliance) EncryptPersonalData(userID uint, dataType GDPRDataType, data interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	encryptedData, err := g.encrypt(dataJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	record := &EncryptedPersonalData{
		UserID:        userID,
		DataType:      string(dataType),
		EncryptedData: encryptedData,
		KeyVersion:    g.keyVersion,
		Algorithm:     "AES-256-GCM",
	}

	return g.db.Create(record).Error
}

// DecryptPersonalData decrypts stored personal data
func (g *GDPRCompliance) DecryptPersonalData(userID uint, dataType GDPRDataType, result interface{}) error {
	var record EncryptedPersonalData
	if err := g.db.Where("user_id = ? AND data_type = ?", userID, string(dataType)).First(&record).Error; err != nil {
		return err
	}

	decryptedData, err := g.decrypt(record.EncryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	return json.Unmarshal(decryptedData, result)
}

// ExportUserData exports all user data for portability requests
func (g *GDPRCompliance) ExportUserData(userID uint) (map[string]interface{}, error) {
	export := make(map[string]interface{})

	// Export consent records
	var consents []ConsentRecord
	g.db.Where("user_id = ?", userID).Find(&consents)
	export["consents"] = consents

	// Export processing records
	var processing []DataProcessingRecord
	g.db.Where("user_id = ?", userID).Find(&processing)
	export["data_processing"] = processing

	// Export data subject requests
	var requests []DataSubjectRequest
	g.db.Where("user_id = ?", userID).Find(&requests)
	export["data_subject_requests"] = requests

	// Export encrypted personal data (decrypted for export)
	var encryptedRecords []EncryptedPersonalData
	g.db.Where("user_id = ?", userID).Find(&encryptedRecords)
	
	personalData := make(map[string]interface{})
	for _, record := range encryptedRecords {
		var data interface{}
		if err := g.DecryptPersonalData(userID, GDPRDataType(record.DataType), &data); err == nil {
			personalData[record.DataType] = data
		}
	}
	export["personal_data"] = personalData

	return export, nil
}

// DeleteUserData securely deletes all user data
func (g *GDPRCompliance) DeleteUserData(userID uint) error {
	tx := g.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete in order of dependencies
	tables := []interface{}{
		&EncryptedPersonalData{},
		&DataSubjectRequest{},
		&DataProcessingRecord{},
		&ConsentRecord{},
	}

	for _, table := range tables {
		if err := tx.Where("user_id = ?", userID).Delete(table).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// encrypt encrypts data using AES-256-GCM
func (g *GDPRCompliance) encrypt(data []byte) (string, error) {
	block, err := aes.NewCipher(g.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-256-GCM
func (g *GDPRCompliance) decrypt(encryptedData string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(g.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GetDataProcessingRegistry returns a registry of all data processing activities
func (g *GDPRCompliance) GetDataProcessingRegistry() ([]map[string]interface{}, error) {
	var records []DataProcessingRecord
	if err := g.db.Find(&records).Error; err != nil {
		return nil, err
	}

	registry := make([]map[string]interface{}, len(records))
	for i, record := range records {
		registry[i] = map[string]interface{}{
			"data_type":        record.DataType,
			"processing_type":  record.ProcessingType,
			"purpose":          record.Purpose,
			"legal_basis":      record.LegalBasis,
			"data_controller":  record.DataController,
			"data_processor":   record.DataProcessor,
			"recipients":       record.Recipients,
			"transfer_country": record.TransferCountry,
			"retention_period": record.RetentionPeriod,
			"processed_at":     record.ProcessedAt,
			"expires_at":       record.ExpiresAt,
		}
	}

	return registry, nil
}

// CleanupExpiredData removes data that has exceeded its retention period
func (g *GDPRCompliance) CleanupExpiredData() error {
	now := time.Now()
	
	// Find expired processing records
	var expiredRecords []DataProcessingRecord
	if err := g.db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Find(&expiredRecords).Error; err != nil {
		return err
	}

	// Delete expired data for each user/data type combination
	for _, record := range expiredRecords {
		// Delete encrypted personal data
		g.db.Where("user_id = ? AND data_type = ?", record.UserID, record.DataType).Delete(&EncryptedPersonalData{})
		
		// Mark processing record as expired
		g.db.Model(&record).Update("processing_type", "expired_deleted")
	}

	return nil
}