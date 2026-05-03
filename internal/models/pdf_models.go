package models

import (
	"database/sql"
	"time"
)

// PDFUpload represents an uploaded PDF file
type PDFUpload struct {
	UploadID              uint64         `gorm:"primaryKey;column:upload_id;autoIncrement" json:"upload_id"`
	JobID                 sql.NullInt64  `gorm:"column:job_id" json:"job_id"`
	DocumentID            sql.NullInt64  `gorm:"column:document_id" json:"document_id"`
	Filename              string         `gorm:"column:filename;not null" json:"filename"`
	OriginalFilename      string         `gorm:"column:original_filename;not null" json:"original_filename"`
	StoredFilename        string         `gorm:"column:stored_filename;not null" json:"stored_filename"`
	FilePath              string         `gorm:"column:file_path;not null" json:"file_path"`
	FileSize              int64          `gorm:"column:file_size;not null" json:"file_size"`
	MimeType              string         `gorm:"column:mime_type;not null" json:"mime_type"`
	FileHash              sql.NullString `gorm:"column:file_hash" json:"file_hash"`
	UploadedBy            sql.NullInt64  `gorm:"column:uploaded_by" json:"uploaded_by"`
	UploadedAt            time.Time      `gorm:"column:uploaded_at;default:CURRENT_TIMESTAMP" json:"uploaded_at"`
	ProcessingStatus      string         `gorm:"column:processing_status;type:enum('pending','processing','completed','failed');default:'pending'" json:"processing_status"`
	ProcessingStartedAt   sql.NullTime   `gorm:"column:processing_started_at" json:"processing_started_at"`
	ProcessingCompletedAt sql.NullTime   `gorm:"column:processing_completed_at" json:"processing_completed_at"`
	ErrorMessage          sql.NullString `gorm:"column:error_message" json:"error_message"`
	IsActive              bool           `gorm:"column:is_active;default:true" json:"is_active"`
}

// TableName specifies the table name for PDFUpload
func (PDFUpload) TableName() string {
	return "pdf_uploads"
}

// PDFExtraction represents OCR extraction results
type PDFExtraction struct {
	ExtractionID     uint64          `gorm:"primaryKey;column:extraction_id;autoIncrement" json:"extraction_id"`
	UploadID         uint64          `gorm:"column:upload_id;not null;uniqueIndex:unique_upload_extraction" json:"upload_id"`
	RawText          sql.NullString  `gorm:"column:raw_text;type:longtext" json:"raw_text"`
	ExtractedData    sql.NullString  `gorm:"column:extracted_data;type:json" json:"extracted_data"` // JSON field
	ConfidenceScore  sql.NullFloat64 `gorm:"column:confidence_score" json:"confidence_score"`
	PageCount        int             `gorm:"column:page_count;default:1" json:"page_count"`
	ExtractionMethod string          `gorm:"column:extraction_method;default:'unipdf'" json:"extraction_method"`
	ExtractedAt      time.Time       `gorm:"column:extracted_at;default:CURRENT_TIMESTAMP" json:"extracted_at"`
	CustomerName     sql.NullString  `gorm:"column:customer_name" json:"customer_name"`
	CustomerID       sql.NullInt64   `gorm:"column:customer_id" json:"customer_id"`
	DocumentDate     sql.NullTime    `gorm:"column:document_date;type:date" json:"document_date"`
	DocumentNumber   sql.NullString  `gorm:"column:document_number" json:"document_number"`
	ParsedTotal      sql.NullFloat64 `gorm:"column:parsed_total" json:"parsed_total"`           // Subtotal before discount
	DiscountAmount   sql.NullFloat64 `gorm:"column:discount_amount" json:"discount_amount"`       // Total discount
	DiscountPercent  sql.NullFloat64 `gorm:"column:discount_percent" json:"discount_percent"`     // Discount percentage
	TotalAmount      sql.NullFloat64 `gorm:"column:total_amount" json:"total_amount"`             // Final amount after discount
	Metadata         sql.NullString  `gorm:"column:metadata;type:json" json:"metadata"` // JSON field
}

// TableName specifies the table name for PDFExtraction
func (PDFExtraction) TableName() string {
	return "pdf_extractions"
}

// PDFExtractionItem represents individual line items extracted from PDFs
type PDFExtractionItem struct {
	ItemID            uint64          `gorm:"primaryKey;column:item_id;autoIncrement" json:"item_id"`
	ExtractionID      uint64          `gorm:"column:extraction_id;not null;index:idx_pdf_items_extraction" json:"extraction_id"`
	LineNumber        sql.NullInt64   `gorm:"column:line_number" json:"line_number"`
	RawProductText    string          `gorm:"column:raw_product_text;not null" json:"raw_product_text"`
	Quantity          sql.NullInt64   `gorm:"column:quantity" json:"quantity"`
	UnitPrice         sql.NullFloat64 `gorm:"column:unit_price" json:"unit_price"`
	LineTotal         sql.NullFloat64 `gorm:"column:line_total" json:"line_total"`
	MappedProductID          sql.NullInt64   `gorm:"column:mapped_product_id;index:idx_pdf_items_product" json:"mapped_product_id"`
	MappedPackageID          sql.NullInt64   `gorm:"column:mapped_package_id;index:idx_pdf_items_package" json:"mapped_package_id"`
	MappedRentalEquipmentID  sql.NullInt64   `gorm:"column:mapped_rental_equipment_id" json:"mapped_rental_equipment_id"`
	MappingConfidence sql.NullFloat64 `gorm:"column:mapping_confidence" json:"mapping_confidence"`
	MappingStatus     string          `gorm:"column:mapping_status;type:enum('pending','auto_mapped','user_confirmed','user_rejected','needs_creation');default:'pending';index:idx_pdf_items_status" json:"mapping_status"`
	UserNotes         sql.NullString  `gorm:"column:user_notes;type:text" json:"user_notes"`
	CreatedAt         time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for PDFExtractionItem
func (PDFExtractionItem) TableName() string {
	return "pdf_extraction_items"
}

// PDFProductMapping represents saved mappings between PDF text and products
type PDFProductMapping struct {
	MappingID       uint64          `gorm:"primaryKey;column:mapping_id;autoIncrement" json:"mapping_id"`
	PDFProductText  string          `gorm:"column:pdf_product_text;not null;uniqueIndex:unique_pdf_text_product;index:idx_pdf_mappings_text" json:"pdf_product_text"`
	NormalizedText  sql.NullString  `gorm:"column:normalized_text;index:idx_pdf_mappings_normalized" json:"normalized_text"`
	ProductID       int             `gorm:"column:product_id;not null;uniqueIndex:unique_pdf_text_product;index:idx_pdf_mappings_product" json:"product_id"`
	MappingType     string          `gorm:"column:mapping_type;type:enum('exact','fuzzy','manual');default:'manual';index:idx_pdf_mappings_type" json:"mapping_type"`
	ConfidenceScore sql.NullFloat64 `gorm:"column:confidence_score" json:"confidence_score"`
	UsageCount      int             `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt      sql.NullTime    `gorm:"column:last_used_at" json:"last_used_at"`
	CreatedBy       sql.NullInt64   `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive        bool            `gorm:"column:is_active;default:true" json:"is_active"`
}

// TableName specifies the table name for PDFProductMapping
func (PDFProductMapping) TableName() string {
	return "pdf_product_mappings"
}

// PDFPackageMapping represents saved mappings between PDF text and packages
type PDFPackageMapping struct {
	MappingID       uint64          `gorm:"primaryKey;column:mapping_id;autoIncrement" json:"mapping_id"`
	PDFPackageText  string          `gorm:"column:pdf_package_text;not null;index:idx_pdf_package_mappings_text" json:"pdf_package_text"`
	NormalizedText  sql.NullString  `gorm:"column:normalized_text;index:idx_pdf_package_mappings_normalized" json:"normalized_text"`
	PackageID       int             `gorm:"column:package_id;not null;index:idx_pdf_package_mappings_package" json:"package_id"`
	MappingType     string          `gorm:"column:mapping_type;type:enum('exact','fuzzy','manual');default:'manual';index:idx_pdf_package_mappings_type" json:"mapping_type"`
	ConfidenceScore sql.NullFloat64 `gorm:"column:confidence_score" json:"confidence_score"`
	UsageCount      int             `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt      sql.NullTime    `gorm:"column:last_used_at" json:"last_used_at"`
	CreatedBy       sql.NullInt64   `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive        bool            `gorm:"column:is_active;default:true" json:"is_active"`
}

// TableName specifies the table name for PDFPackageMapping
func (PDFPackageMapping) TableName() string {
	return "pdf_package_mappings"
}

// PDFRentalMapping represents saved mappings between PDF text and rental equipment
type PDFRentalMapping struct {
	MappingID        uint64          `gorm:"primaryKey;column:mapping_id;autoIncrement" json:"mapping_id"`
	PDFRentalText    string          `gorm:"column:pdf_rental_text;not null;uniqueIndex:unique_pdf_rental_text" json:"pdf_rental_text"`
	NormalizedText   sql.NullString  `gorm:"column:normalized_text;index:idx_pdf_rental_mappings_normalized" json:"normalized_text"`
	RentalEquipmentID int            `gorm:"column:rental_equipment_id;not null" json:"rental_equipment_id"`
	MappingType      string          `gorm:"column:mapping_type;default:'manual'" json:"mapping_type"`
	ConfidenceScore  sql.NullFloat64 `gorm:"column:confidence_score" json:"confidence_score"`
	UsageCount       int             `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt       sql.NullTime    `gorm:"column:last_used_at" json:"last_used_at"`
	CreatedBy        sql.NullInt64   `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive         bool            `gorm:"column:is_active;default:true" json:"is_active"`
}

func (PDFRentalMapping) TableName() string {
	return "pdf_rental_mappings"
}

// PDFCustomerMapping represents saved mappings between PDF text and customers
type PDFCustomerMapping struct {
	MappingID       uint64          `gorm:"primaryKey;column:mapping_id;autoIncrement" json:"mapping_id"`
	PDFCustomerText string          `gorm:"column:pdf_customer_text;not null;index:idx_pdf_customer_mappings_text" json:"pdf_customer_text"`
	NormalizedText  sql.NullString  `gorm:"column:normalized_text;index:idx_pdf_customer_mappings_normalized" json:"normalized_text"`
	CustomerID      int             `gorm:"column:customer_id;not null;index:idx_pdf_customer_mappings_customer" json:"customer_id"`
	MappingType     string          `gorm:"column:mapping_type;type:enum('exact','fuzzy','manual');default:'manual'" json:"mapping_type"`
	ConfidenceScore sql.NullFloat64 `gorm:"column:confidence_score" json:"confidence_score"`
	UsageCount      int             `gorm:"column:usage_count;default:0" json:"usage_count"`
	LastUsedAt      sql.NullTime    `gorm:"column:last_used_at" json:"last_used_at"`
	CreatedBy       sql.NullInt64   `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	IsActive        bool            `gorm:"column:is_active;default:true" json:"is_active"`
}

// TableName specifies the table name for PDFCustomerMapping
func (PDFCustomerMapping) TableName() string {
	return "pdf_customer_mappings"
}

// PDFExtractionResponse is the API response structure for extracted data
type PDFExtractionResponse struct {
	UploadID        uint64                     `json:"upload_id"`
	ExtractionID    uint64                     `json:"extraction_id"`
	CustomerName    string                     `json:"customer_name,omitempty"`
	CustomerID      *int                       `json:"customer_id,omitempty"`
	DocumentNumber  string                     `json:"document_number,omitempty"`
	DocumentDate    string                     `json:"document_date,omitempty"`
	StartDate       string                     `json:"start_date,omitempty"`
	EndDate         string                     `json:"end_date,omitempty"`
	TotalAmount     float64                    `json:"total_amount,omitempty"`
	DiscountAmount  float64                    `json:"discount_amount,omitempty"`
	Items           []PDFExtractionItem        `json:"items"`
	RawText         string                     `json:"raw_text,omitempty"`
	ConfidenceScore float64                    `json:"confidence_score,omitempty"`
	Suggestions     []ProductMappingSuggestion `json:"suggestions,omitempty"`
}

// ProductMappingSuggestion represents a suggested product mapping
type ProductMappingSuggestion struct {
	ItemID           uint64   `json:"item_id"`
	RawProductText   string   `json:"raw_product_text"`
	SuggestedProduct *Product `json:"suggested_product,omitempty"`
	Confidence       float64  `json:"confidence"`
	MappingType      string   `json:"mapping_type"` // 'exact', 'fuzzy', 'previous', 'package'
	PackageID        *int     `json:"package_id,omitempty"`
	PackageCode      string   `json:"package_code,omitempty"`
	PackageName      string   `json:"package_name,omitempty"`
	PackagePrice     *float64 `json:"package_price,omitempty"`
}

// PDFMappingEvent represents a single mapping action (manual or persisted)
type PDFMappingEvent struct {
	EventID        uint64         `gorm:"primaryKey;column:event_id;autoIncrement" json:"event_id"`
	ExtractionID   sql.NullInt64  `gorm:"column:extraction_id" json:"extraction_id"`
	ItemID         sql.NullInt64  `gorm:"column:item_id" json:"item_id"`
	PDFProductText string         `gorm:"column:pdf_product_text;not null" json:"pdf_product_text"`
	NormalizedText sql.NullString `gorm:"column:normalized_text" json:"normalized_text"`
	ProductID      sql.NullInt64  `gorm:"column:product_id" json:"product_id"`
	PackageID      sql.NullInt64  `gorm:"column:package_id" json:"package_id"`
	CreatedBy      sql.NullInt64  `gorm:"column:created_by" json:"created_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName specifies the table name for PDFMappingEvent
func (PDFMappingEvent) TableName() string {
	return "pdf_mapping_events"
}
