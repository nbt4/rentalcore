package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services/pdf"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PDFHandler handles PDF upload and processing requests
type PDFHandler struct {
	DB              *gorm.DB
	Extractor       *pdf.PDFExtractor
	Mapper          *pdf.ProductMapper
	PackageMapper   *pdf.PackageMapper
	CustomerMapper  *pdf.CustomerMapper
	JobHandler      *JobHandler
	AttachmentRepo  *repository.JobAttachmentRepository
	JobPackageRepo  *repository.JobPackageRepository
	DocumentHandler *DocumentHandler
	attachmentDir   string
}

type duplicateJobMatch struct {
	JobID       uint   `json:"job_id"`
	JobCode     string `json:"job_code"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	Status      string `json:"status,omitempty"`
	DeviceCount int    `json:"device_count"`
	JobsURL     string `json:"jobs_url"`
}

func applySuggestionToNewItem(item *models.PDFExtractionItem, suggestion *models.ProductMappingSuggestion) {
	if item == nil || suggestion == nil {
		return
	}

	if suggestion.PackageID != nil && *suggestion.PackageID > 0 {
		item.MappedPackageID = sql.NullInt64{Int64: int64(*suggestion.PackageID), Valid: true}
		item.MappedProductID = sql.NullInt64{}
	} else if suggestion.SuggestedProduct != nil {
		item.MappedProductID = sql.NullInt64{Int64: int64(suggestion.SuggestedProduct.ProductID), Valid: true}
		item.MappedPackageID = sql.NullInt64{}
	} else {
		return
	}

	if suggestion.Confidence > 0 {
		item.MappingConfidence = sql.NullFloat64{Float64: suggestion.Confidence, Valid: true}
	}

	if suggestion.Confidence >= 80 {
		item.MappingStatus = "auto_mapped"
	} else if item.MappingStatus == "" {
		item.MappingStatus = "pending"
	}
}

func ensurePackageMappingSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if err := ensurePackageMappingColumn(sqlDB); err != nil {
		return err
	}
	if err := ensurePackageMappingIndex(sqlDB); err != nil {
		return err
	}
	return ensurePackageMappingFK(sqlDB)
}

func ensurePackageMappingColumn(db *sql.DB) error {
	const query = `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'pdf_extraction_items'
		  AND column_name = 'mapped_package_id'
	`
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE pdf_extraction_items ADD COLUMN mapped_package_id INTEGER`)
	return err
}

func ensurePackageMappingIndex(db *sql.DB) error {
	const query = `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND tablename = 'pdf_extraction_items'
		  AND indexname = 'idx_pdf_items_package'
	`
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.Exec(`CREATE INDEX idx_pdf_items_package ON pdf_extraction_items (mapped_package_id)`)
	return err
}

func ensurePackageMappingFK(db *sql.DB) error {
	const query = `
		SELECT COUNT(*)
		FROM information_schema.table_constraints
		WHERE table_schema = 'public'
		  AND table_name = 'pdf_extraction_items'
		  AND constraint_name = 'fk_pdf_items_package'
	`
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.Exec(`
		ALTER TABLE pdf_extraction_items
		ADD CONSTRAINT fk_pdf_items_package
			FOREIGN KEY (mapped_package_id)
			REFERENCES product_packages(package_id)
			ON DELETE SET NULL
	`)
	return err
}

func buildSuggestionUpdates(suggestion *models.ProductMappingSuggestion, status string) map[string]interface{} {
	if suggestion == nil {
		return nil
	}

	updates := map[string]interface{}{
		"mapping_status": status,
	}

	if suggestion.Confidence > 0 {
		updates["mapping_confidence"] = suggestion.Confidence
	}

	if suggestion.PackageID != nil && *suggestion.PackageID > 0 {
		updates["mapped_package_id"] = *suggestion.PackageID
		updates["mapped_product_id"] = nil
	} else if suggestion.SuggestedProduct != nil {
		updates["mapped_product_id"] = suggestion.SuggestedProduct.ProductID
		updates["mapped_package_id"] = nil
	} else {
		return nil
	}

	return updates
}

func getItemQuantity(item *models.PDFExtractionItem) int {
	if item == nil {
		return 0
	}
	if item.Quantity.Valid && item.Quantity.Int64 > 0 {
		return int(item.Quantity.Int64)
	}
	return 1
}

func resolveLinePricing(item *models.PDFExtractionItem, qty int) (float64, bool) {
	if item == nil || qty <= 0 {
		return 0, false
	}

	if item.LineTotal.Valid {
		return item.LineTotal.Float64, true
	}

	if item.UnitPrice.Valid {
		return item.UnitPrice.Float64 * float64(qty), true
	}

	return 0, false
}

type packageSummary struct {
	PackageID   int      `json:"package_id"`
	PackageCode string   `json:"package_code"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
}

func sanitizePackage(pkg *models.ProductPackage) *packageSummary {
	if pkg == nil {
		return nil
	}
	summary := &packageSummary{
		PackageID:   pkg.PackageID,
		PackageCode: pkg.PackageCode,
		Name:        pkg.Name,
	}
	if pkg.Description.Valid {
		summary.Description = pkg.Description.String
	}
	if pkg.Price.Valid {
		val := pkg.Price.Float64
		summary.Price = &val
	}
	return summary
}

// NewPDFHandler creates a new PDF handler
func NewPDFHandler(db *gorm.DB, uploadDir string, jobHandler *JobHandler, attachmentRepo *repository.JobAttachmentRepository, aliasCache *pdf.PackageAliasCache, documentHandler *DocumentHandler) *PDFHandler {
	attachmentDir := filepath.Join(uploadDir, "job_attachments")
	if err := os.MkdirAll(attachmentDir, 0755); err != nil {
		log.Printf("warning: failed to ensure attachment directory %s: %v", attachmentDir, err)
	}

	if err := ensurePackageMappingSchema(db); err != nil {
		log.Printf("warning: failed to ensure PDF package mapping schema: %v", err)
	}

	// Initialize repositories
	dbWrapper := &repository.Database{DB: db}
	jobPackageRepo := repository.NewJobPackageRepository(dbWrapper)

	return &PDFHandler{
		DB:              db,
		Extractor:       pdf.NewPDFExtractor(uploadDir),
		Mapper:          pdf.NewProductMapper(db, aliasCache),
		PackageMapper:   pdf.NewPackageMapper(db),
		CustomerMapper:  pdf.NewCustomerMapper(db),
		JobHandler:      jobHandler,
		AttachmentRepo:  attachmentRepo,
		JobPackageRepo:  jobPackageRepo,
		DocumentHandler: documentHandler,
		attachmentDir:   attachmentDir,
	}
}

// UploadPDF handles PDF file upload
// POST /api/v1/pdf/upload
func (h *PDFHandler) UploadPDF(c *gin.Context) {
	// Get job ID if provided
	jobIDStr := c.PostForm("job_id")
	var jobID sql.NullInt64
	if jobIDStr != "" {
		id, err := strconv.ParseInt(jobIDStr, 10, 64)
		if err == nil {
			jobID = sql.NullInt64{Int64: id, Valid: true}
		}
	}

	// Get uploaded file
	file, err := c.FormFile("pdf")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file type
	if file.Header.Get("Content-Type") != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	// Save file
	upload, err := h.Extractor.SaveUploadedFile(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}

	// Set job ID if provided
	upload.JobID = jobID

	// Get user ID from session
	if userID, exists := c.Get("userid"); exists {
		if uid, ok := userID.(int64); ok {
			upload.UploadedBy = sql.NullInt64{Int64: uid, Valid: true}
		}
	}

	// Save upload record to database
	if err := h.DB.Create(upload).Error; err != nil {
		log.Printf("ERROR: Failed to save upload record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save upload record: %v", err)})
		return
	}

	if upload.JobID.Valid {
		h.attachUploadToJob(upload, uint(upload.JobID.Int64))
	}

	// Start processing asynchronously
	go h.processUploadAsync(upload.UploadID)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"upload_id": upload.UploadID,
		"message":   "PDF uploaded successfully, processing started",
	})
}

// processUploadAsync processes the PDF asynchronously
func (h *PDFHandler) processUploadAsync(uploadID uint64) {
	// Update status to processing
	h.DB.Model(&models.PDFUpload{}).Where("upload_id = ?", uploadID).Updates(map[string]interface{}{
		"processing_status":     "processing",
		"processing_started_at": time.Now(),
	})

	// Get upload record
	var upload models.PDFUpload
	if err := h.DB.First(&upload, uploadID).Error; err != nil {
		h.markProcessingFailed(uploadID, fmt.Sprintf("Upload not found: %v", err))
		return
	}

	// Extract text
	rawText, err := h.Extractor.ExtractText(upload.FilePath)
	if err != nil {
		h.markProcessingFailed(uploadID, fmt.Sprintf("Text extraction failed: %v", err))
		return
	}

	// Parse invoice data using Python parser
	parsedDoc, err := h.Extractor.ParseDocumentIntelligently(rawText)
	if err != nil {
		h.markProcessingFailed(uploadID, fmt.Sprintf("Data parsing failed: %v", err))
		return
	}

	// Attempt customer auto-mapping
	var customerID *int
	if parsedDoc.CustomerName != "" && h.CustomerMapper != nil {
		if _, customer, confidence, err := h.CustomerMapper.FindBestMatch(parsedDoc.CustomerName); err == nil && customer != nil && confidence >= 60 {
			id := int(customer.CustomerID)
			customerID = &id
		}
	}

	// Convert to JSON
	extractedDataJSON, err := json.Marshal(parsedDoc)
	if err != nil {
		h.markProcessingFailed(uploadID, fmt.Sprintf("JSON conversion failed: %v", err))
		return
	}

	// Create extraction record
	extraction := models.PDFExtraction{
		UploadID:         uploadID,
		RawText:          sql.NullString{String: rawText, Valid: true},
		ExtractedData:    sql.NullString{String: string(extractedDataJSON), Valid: true},
		ConfidenceScore:  sql.NullFloat64{Float64: parsedDoc.ConfidenceScore, Valid: true},
		PageCount:        1, // TODO: Get actual page count
		ExtractionMethod: "python_parser",
		CustomerName:     sql.NullString{String: parsedDoc.CustomerName, Valid: parsedDoc.CustomerName != ""},
		DocumentNumber:   sql.NullString{String: parsedDoc.DocumentNumber, Valid: parsedDoc.DocumentNumber != ""},
		ParsedTotal:      sql.NullFloat64{Float64: parsedDoc.ParsedTotal, Valid: parsedDoc.ParsedTotal > 0},
		DiscountAmount:   sql.NullFloat64{Float64: parsedDoc.DiscountAmount, Valid: parsedDoc.DiscountAmount > 0},
		DiscountPercent:  sql.NullFloat64{Float64: parsedDoc.DiscountPercent, Valid: parsedDoc.DiscountPercent > 0},
		TotalAmount:      sql.NullFloat64{Float64: parsedDoc.TotalAmount, Valid: parsedDoc.TotalAmount > 0},
	}
	if customerID != nil && *customerID > 0 {
		extraction.CustomerID = sql.NullInt64{Int64: int64(*customerID), Valid: true}
	}

	if !parsedDoc.DocumentDate.IsZero() {
		extraction.DocumentDate = sql.NullTime{Time: parsedDoc.DocumentDate, Valid: true}
	}

	// Store metadata from Python parser; seed start_date from DocumentDate if not set
	meta := parsedDoc.Metadata
	if meta == nil {
		meta = make(map[string]interface{})
	}
	if _, hasStart := meta["start_date"]; !hasStart && !parsedDoc.DocumentDate.IsZero() {
		meta["start_date"] = parsedDoc.DocumentDate.Format(time.RFC3339)
	}
	if metaBytes, err := json.Marshal(meta); err == nil {
		extraction.Metadata = sql.NullString{String: string(metaBytes), Valid: true}
	}

	// Save extraction
	if err := h.DB.Create(&extraction).Error; err != nil {
		h.markProcessingFailed(uploadID, fmt.Sprintf("Failed to save extraction: %v", err))
		return
	}

	// Create extraction items
	for _, item := range parsedDoc.Items {
		extractionItem := models.PDFExtractionItem{
			ExtractionID:   extraction.ExtractionID,
			LineNumber:     sql.NullInt64{Int64: int64(item.LineNumber), Valid: true},
			RawProductText: item.ProductName,
			Quantity:       sql.NullInt64{Int64: int64(item.Quantity), Valid: true},
			UnitPrice:      sql.NullFloat64{Float64: item.UnitPrice, Valid: item.UnitPrice >= 0},
			LineTotal:      sql.NullFloat64{Float64: item.LineTotal, Valid: true},
			MappingStatus:  "pending",
		}

		// First check for saved package mapping
		var packageMatch *models.ProductPackage
		if h.PackageMapper != nil {
			packageMatch, _ = h.PackageMapper.LookupSavedMapping(item.ProductName)
		}

		// If package found, use it with high confidence
		if packageMatch != nil {
			extractionItem.MappedPackageID = sql.NullInt64{Int64: int64(packageMatch.PackageID), Valid: true}
			extractionItem.MappingConfidence = sql.NullFloat64{Float64: 100.0, Valid: true}
			extractionItem.MappingStatus = "auto_mapped"
		} else {
			// Try to find product mapping
			if suggestion, err := h.Mapper.FindBestMatch(item.ProductName); err == nil && suggestion != nil {
				applySuggestionToNewItem(&extractionItem, suggestion)
			}
		}

		h.DB.Create(&extractionItem)
	}

	// Mark as completed
	h.DB.Model(&models.PDFUpload{}).Where("upload_id = ?", uploadID).Updates(map[string]interface{}{
		"processing_status":       "completed",
		"processing_completed_at": time.Now(),
	})
}

// markProcessingFailed marks upload as failed
func (h *PDFHandler) markProcessingFailed(uploadID uint64, errorMsg string) {
	h.DB.Model(&models.PDFUpload{}).Where("upload_id = ?", uploadID).Updates(map[string]interface{}{
		"processing_status":       "failed",
		"processing_completed_at": time.Now(),
		"error_message":           errorMsg,
	})
}

// GetExtractionResult retrieves extraction results
// GET /api/v1/pdf/extraction/:upload_id
func (h *PDFHandler) GetExtractionResult(c *gin.Context) {
	uploadID := c.Param("upload_id")

	var upload models.PDFUpload
	if err := h.DB.First(&upload, uploadID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}

	var extraction models.PDFExtraction
	if err := h.DB.Where("upload_id = ?", uploadID).First(&extraction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"})
		return
	}

	var items []models.PDFExtractionItem
	h.DB.Where("extraction_id = ?", extraction.ExtractionID).Find(&items)

	// Resolve product/package names for mapped items
	type ItemWithName struct {
		models.PDFExtractionItem
		MappedName string `json:"mapped_name,omitempty"`
	}
	productIDs, packageIDs := []int{}, []int{}
	for _, it := range items {
		if it.MappedProductID.Valid { productIDs = append(productIDs, int(it.MappedProductID.Int64)) }
		if it.MappedPackageID.Valid { packageIDs = append(packageIDs, int(it.MappedPackageID.Int64)) }
	}
	productNames := make(map[int]string)
	if len(productIDs) > 0 {
		var prods []models.Product
		h.DB.Select("productid, name").Where("productid IN ?", productIDs).Find(&prods)
		for _, p := range prods { productNames[int(p.ProductID)] = p.Name }
	}
	packageNames := make(map[int]string)
	if len(packageIDs) > 0 {
		var pkgs []models.ProductPackage
		h.DB.Select("package_id, name").Where("package_id IN ?", packageIDs).Find(&pkgs)
		for _, p := range pkgs { packageNames[p.PackageID] = p.Name }
	}
	enrichedItems := make([]ItemWithName, 0, len(items))
	for _, it := range items {
		iwn := ItemWithName{PDFExtractionItem: it}
		if it.MappedProductID.Valid {
			iwn.MappedName = productNames[int(it.MappedProductID.Int64)]
		} else if it.MappedPackageID.Valid {
			iwn.MappedName = packageNames[int(it.MappedPackageID.Int64)]
		}
		enrichedItems = append(enrichedItems, iwn)
	}

	// Build response
	type ExtractionResponseWithNames struct {
		UploadID        uint64         `json:"upload_id"`
		ExtractionID    uint64         `json:"extraction_id"`
		CustomerName    string         `json:"customer_name,omitempty"`
		CustomerID      *int           `json:"customer_id,omitempty"`
		DocumentNumber  string         `json:"document_number,omitempty"`
		DocumentDate    string         `json:"document_date,omitempty"`
		StartDate       string         `json:"start_date,omitempty"`
		EndDate         string         `json:"end_date,omitempty"`
		TotalAmount     float64        `json:"total_amount,omitempty"`
		DiscountAmount  float64        `json:"discount_amount,omitempty"`
		RawText         string         `json:"raw_text,omitempty"`
		ConfidenceScore float64        `json:"confidence_score,omitempty"`
		Items           []ItemWithName `json:"items"`
	}
	response := ExtractionResponseWithNames{
		UploadID:     upload.UploadID,
		ExtractionID: extraction.ExtractionID,
		Items:        enrichedItems,
	}

	if extraction.CustomerName.Valid {
		response.CustomerName = extraction.CustomerName.String
	}
	if extraction.CustomerID.Valid {
		customerID := int(extraction.CustomerID.Int64)
		response.CustomerID = &customerID
	}
	if extraction.DocumentNumber.Valid {
		response.DocumentNumber = extraction.DocumentNumber.String
	}
	if extraction.DocumentDate.Valid {
		response.DocumentDate = extraction.DocumentDate.Time.Format("2006-01-02")
	}
	if extraction.Metadata.Valid {
		var meta map[string]string
		if err := json.Unmarshal([]byte(extraction.Metadata.String), &meta); err == nil {
			formatMetaDate := func(s string) string {
				for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00", "2006-01-02"} {
					if t, err := time.Parse(layout, s); err == nil {
						return t.Format("2006-01-02")
					}
				}
				return "" // unparseable → omit, don't pass garbage to frontend
			}
			if start, ok := meta["start_date"]; ok {
				response.StartDate = formatMetaDate(start)
			}
			if end, ok := meta["end_date"]; ok {
				response.EndDate = formatMetaDate(end)
			}
		}
	}
	if extraction.TotalAmount.Valid {
		response.TotalAmount = extraction.TotalAmount.Float64
	}
	if extraction.DiscountAmount.Valid {
		response.DiscountAmount = extraction.DiscountAmount.Float64
	}
	if extraction.RawText.Valid {
		response.RawText = extraction.RawText.String
	}
	if extraction.ConfidenceScore.Valid {
		response.ConfidenceScore = extraction.ConfidenceScore.Float64
	}

	c.JSON(http.StatusOK, response)
}

// SaveProductMapping saves a manual product mapping
// POST /api/v1/pdf/mapping
func (h *PDFHandler) SaveProductMapping(c *gin.Context) {
	var req struct {
		PDFText   string `json:"pdf_text" binding:"required"`
		ProductID int    `json:"product_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := int64(1) // TODO: Get from session
	if uid, exists := c.Get("userid"); exists {
		if id, ok := uid.(int64); ok {
			userID = id
		}
	}

	if err := h.Mapper.SaveMapping(req.PDFText, req.ProductID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Mapping saved successfully"})
}

// GetProductSuggestions gets product suggestions for PDF text
// GET /api/v1/pdf/suggestions?text=...
func (h *PDFHandler) GetProductSuggestions(c *gin.Context) {
	productText := c.Query("text")
	if productText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Text parameter required"})
		return
	}

	suggestions, err := h.Mapper.FindSimilarProducts(productText, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find suggestions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// UpdateItemMapping updates the product mapping for an extraction item
// PUT /api/v1/pdf/items/:item_id/mapping
func (h *PDFHandler) UpdateItemMapping(c *gin.Context) {
	itemID := c.Param("item_id")

	var req struct {
		ProductID *int   `json:"product_id"`
		PackageID *int   `json:"package_id"`
		Status    string `json:"status"` // 'user_confirmed', 'user_rejected', etc.
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if (req.ProductID == nil || *req.ProductID <= 0) && (req.PackageID == nil || *req.PackageID <= 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid product_id or package_id is required"})
		return
	}

	status := req.Status
	if status == "" {
		status = "user_confirmed"
	}

	updates := map[string]interface{}{
		"mapping_status":     status,
		"mapping_confidence": 100.0,
	}

	if req.PackageID != nil && *req.PackageID > 0 {
		updates["mapped_package_id"] = *req.PackageID
		updates["mapped_product_id"] = nil
	} else if req.ProductID != nil && *req.ProductID > 0 {
		updates["mapped_product_id"] = *req.ProductID
		updates["mapped_package_id"] = nil
	}

	if err := h.DB.Model(&models.PDFExtractionItem{}).Where("item_id = ?", itemID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Mapping updated successfully"})
}

// ShowReviewScreen displays the PDF extraction review screen
// GET /pdf/review/:upload_id
func (h *PDFHandler) ShowReviewScreen(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// Get upload record
	var upload models.PDFUpload
	if err := h.DB.First(&upload, uploadID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Upload not found",
		})
		return
	}

	// Get extraction record
	var extraction models.PDFExtraction
	if err := h.DB.Where("upload_id = ?", uploadID).First(&extraction).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Extraction not found - processing may still be in progress",
		})
		return
	}

	// Get extraction items
	var items []models.PDFExtractionItem
	h.DB.Where("extraction_id = ?", extraction.ExtractionID).Order("line_number").Find(&items)

	itemDiscounts := make(map[uint64]float64)
	for _, item := range items {
		if discount := calculateExtractionItemDiscount(&item); discount > 0 {
			itemDiscounts[item.ItemID] = discount
		}
	}

	mappedProducts := make(map[uint64]models.Product)
	productIDs := make([]int64, 0)
	for _, item := range items {
		if item.MappedProductID.Valid {
			productIDs = append(productIDs, item.MappedProductID.Int64)
		}
	}

	if len(productIDs) > 0 {
		var products []models.Product
		if err := h.DB.Where("productID IN ?", productIDs).Find(&products).Error; err == nil {
			productLookup := make(map[int64]models.Product, len(products))
			for _, product := range products {
				productLookup[int64(product.ProductID)] = product
			}
			for _, item := range items {
				if item.MappedProductID.Valid {
					if product, ok := productLookup[item.MappedProductID.Int64]; ok {
						mappedProducts[item.ItemID] = product
					}
				}
			}
		}
	}

	// Prepare response data
	data := gin.H{
		"upload":         upload,
		"extraction":     extraction,
		"items":          items,
		"itemDiscounts":  itemDiscounts,
		"mappedProducts": mappedProducts,
		"pageTitle":      "PDF Extraction Review",
	}

	totalItems := len(items)
	mappedCount := 0
	for _, item := range items {
		if item.MappingStatus == "auto_mapped" || item.MappingStatus == "user_confirmed" {
			mappedCount++
			continue
		}
		if item.MappedProductID.Valid && item.MappingStatus == "pending" {
			mappedCount++
		}
	}
	pendingCount := totalItems - mappedCount
	mappedPercent := 0
	if totalItems > 0 {
		mappedPercent = int(math.Round(float64(mappedCount) / float64(totalItems) * 100))
	}

	data["totalItems"] = totalItems
	data["mappedCount"] = mappedCount
	data["pendingCount"] = pendingCount
	data["mappedPercent"] = mappedPercent

	// Add optional fields
	if extraction.CustomerName.Valid {
		data["customerName"] = extraction.CustomerName.String
	}
	if extraction.DocumentNumber.Valid {
		data["documentNumber"] = extraction.DocumentNumber.String
	}
	if extraction.DocumentDate.Valid {
		data["documentDate"] = extraction.DocumentDate.Time.Format("2006-01-02")
	}
	if extraction.Metadata.Valid {
		var meta map[string]string
		if err := json.Unmarshal([]byte(extraction.Metadata.String), &meta); err == nil {
			if startStr, ok := meta["start_date"]; ok {
				if t, err := time.Parse(time.RFC3339, startStr); err == nil {
					data["startdate"] = t
				}
			}
			if endStr, ok := meta["end_date"]; ok {
				if t, err := time.Parse(time.RFC3339, endStr); err == nil {
					data["enddate"] = t
				}
			}
		}
	}
	if extraction.ParsedTotal.Valid {
		data["parsedTotal"] = extraction.ParsedTotal.Float64
	}
	if extraction.DiscountAmount.Valid {
		data["discountAmount"] = extraction.DiscountAmount.Float64
	}
	if extraction.DiscountPercent.Valid {
		data["discountPercent"] = extraction.DiscountPercent.Float64
	}
	if extraction.TotalAmount.Valid {
		data["totalAmount"] = extraction.TotalAmount.Float64
	}

	c.HTML(http.StatusOK, "pdf_review.html", data)
}

// ShowMappingScreen displays the PDF product mapping screen
// GET /pdf/mapping/:extraction_id
func (h *PDFHandler) ShowMappingScreen(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	// Get extraction record
	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Extraction not found",
		})
		return
	}

	// Get upload record
	var upload models.PDFUpload
	h.DB.First(&upload, extraction.UploadID)

	// Get extraction items with product/package mappings
	var items []models.PDFExtractionItem
	h.DB.Where("extraction_id = ?", extractionID).Order("line_number").Find(&items)
	productCounts, mappedProductItems, pendingProductItems := h.summarizeExtractionItems(items)

	packageLookup := make(map[int64]*models.ProductPackage)
	packageIDs := make([]int64, 0)
	seenPackages := make(map[int64]struct{})
	for _, item := range items {
		if item.MappedPackageID.Valid {
			pkgID := item.MappedPackageID.Int64
			if _, exists := seenPackages[pkgID]; !exists {
				seenPackages[pkgID] = struct{}{}
				packageIDs = append(packageIDs, pkgID)
			}
		}
	}
	if len(packageIDs) > 0 {
		var packages []models.ProductPackage
		if err := h.DB.Where("package_id IN ?", packageIDs).Find(&packages).Error; err == nil {
			for idx := range packages {
				packageLookup[int64(packages[idx].PackageID)] = &packages[idx]
			}
		}
	}

	// For each item, get mapping suggestions
	itemsWithSuggestions := make([]gin.H, 0, len(items))
	for _, item := range items {
		suggestions, _ := h.Mapper.FindSimilarProducts(item.RawProductText, 5)

		itemData := gin.H{
			"item":        item,
			"suggestions": suggestions,
		}

		// Add mapped product/package if exists
		if item.MappedProductID.Valid {
			var product models.Product
			if err := h.DB.First(&product, item.MappedProductID.Int64).Error; err == nil {
				itemData["mappedProduct"] = product
			}
		} else if item.MappedPackageID.Valid {
			if pkg := packageLookup[item.MappedPackageID.Int64]; pkg != nil {
				itemData["mappedPackage"] = pkg
			}
		}

		itemsWithSuggestions = append(itemsWithSuggestions, itemData)
	}

	var meta map[string]string
	if extraction.Metadata.Valid {
		_ = json.Unmarshal([]byte(extraction.Metadata.String), &meta)
	}

	parseMetaDate := func(key string) *time.Time {
		if meta == nil {
			return nil
		}
		if value, ok := meta[key]; ok && value != "" {
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				return &t
			}
		}
		return nil
	}

	startDate := parseMetaDate("start_date")
	endDate := parseMetaDate("end_date")

	if startDate == nil && extraction.DocumentDate.Valid {
		value := extraction.DocumentDate.Time
		startDate = &value
	}

	// Determine discount type from extraction
	discountType := "amount"

	// If discount_percent is present, use percent type
	if extraction.DiscountPercent.Valid && extraction.DiscountPercent.Float64 > 0 {
		discountType = "percent"
	} else if extraction.DiscountAmount.Valid && extraction.DiscountAmount.Float64 > 0 {
		discountType = "amount"
	}

	// Override with metadata if explicitly set
	if extraction.Metadata.Valid {
		var meta map[string]string
		if err := json.Unmarshal([]byte(extraction.Metadata.String), &meta); err == nil {
			if dt := strings.TrimSpace(meta["discount_type"]); dt != "" {
				discountType = dt
			}
		}
	}

	totalItems := len(items)
	mappedCount := 0
	for _, item := range items {
		if item.MappingStatus == "auto_mapped" || item.MappingStatus == "user_confirmed" {
			mappedCount++
			continue
		}
		if item.MappedProductID.Valid && item.MappingStatus == "pending" {
			mappedCount++
		}
	}
	pendingCount := totalItems - mappedCount
	mappedPercent := 0
	if totalItems > 0 {
		mappedPercent = int(math.Round(float64(mappedCount) / float64(totalItems) * 100))
	}

	currentCustomerID := uint(0)
	if extraction.CustomerID.Valid && extraction.CustomerID.Int64 > 0 {
		currentCustomerID = uint(extraction.CustomerID.Int64)
	}

	data := gin.H{
		"extraction":    extraction,
		"upload":        upload,
		"items":         itemsWithSuggestions,
		"pageTitle":     "PDF Product Mapping",
		"startdate":     formatDateInput(startDate),
		"enddate":       formatDateInput(endDate),
		"discountType":  discountType,
		"totalItems":    totalItems,
		"mappedCount":   mappedCount,
		"pendingCount":  pendingCount,
		"mappedPercent": mappedPercent,
	}

	if extraction.ParsedTotal.Valid {
		data["parsedTotal"] = extraction.ParsedTotal.Float64
	}

	if extraction.DiscountAmount.Valid {
		data["discountAmount"] = extraction.DiscountAmount.Float64
	}

	if extraction.DiscountPercent.Valid {
		data["discountPercent"] = extraction.DiscountPercent.Float64
	}

	// totalAmount is the final amount AFTER discount (Gesamtbetrag)
	// parsedTotal is the subtotal BEFORE discount (Zwischensumme)
	if extraction.TotalAmount.Valid {
		data["totalAmount"] = extraction.TotalAmount.Float64
		// Net amount is the final amount (already includes discount)
		data["netAmount"] = extraction.TotalAmount.Float64
	} else if extraction.ParsedTotal.Valid {
		// Fallback: calculate from parsed_total if total_amount not available
		net := extraction.ParsedTotal.Float64
		if extraction.DiscountAmount.Valid {
			net -= extraction.DiscountAmount.Float64
			if net < 0 {
				net = 0
			}
		}
		data["netAmount"] = net
		data["totalAmount"] = net
	}

	if extraction.CustomerName.Valid {
		data["extractedCustomerName"] = extraction.CustomerName.String
	}
	if currentCustomerID > 0 {
		data["selectedCustomerID"] = currentCustomerID
		var customer models.Customer
		if err := h.DB.First(&customer, currentCustomerID).Error; err == nil {
			data["selectedCustomerName"] = customer.GetDisplayName()
		}
	}

	canRunDuplicateCheck := mappedProductItems > 0 && currentCustomerID > 0
	duplicateMatches := []duplicateJobMatch{}
	if canRunDuplicateCheck {
		excludeJobID := uint(0)
		if upload.JobID.Valid && upload.JobID.Int64 > 0 {
			excludeJobID = uint(upload.JobID.Int64)
		}
		if matches, err := h.detectDuplicateJobs(currentCustomerID, productCounts, excludeJobID); err != nil {
			log.Printf("warning: duplicate detection failed: %v", err)
		} else {
			duplicateMatches = matches
		}
	}

	data["duplicatePendingItems"] = pendingProductItems
	data["duplicateCheckReady"] = canRunDuplicateCheck

	bootstrap := map[string]interface{}{
		"matches": duplicateMatches,
		"ready":   canRunDuplicateCheck,
		"pending": pendingProductItems,
		"checked": canRunDuplicateCheck,
	}
	bootstrapJSON := `{"matches":[],"ready":false,"pending":0,"checked":false}`
	if payload, err := json.Marshal(bootstrap); err == nil {
		bootstrapJSON = string(payload)
	}
	data["duplicateCheckJSON"] = template.JS(bootstrapJSON)

	if prefill := h.buildCustomerPrefill(&extraction); prefill != nil {
		data["hasCustomerPrefill"] = true
		if raw, err := json.Marshal(prefill); err == nil {
			data["customerPrefillJSON"] = template.JS(raw)
		} else {
			data["customerPrefillJSON"] = template.JS("null")
		}
	} else {
		data["customerPrefillJSON"] = template.JS("null")
		data["hasCustomerPrefill"] = false
	}

	c.HTML(http.StatusOK, "pdf_mapping.html", data)
}

// RunAutoMapping runs automatic product mapping for an extraction
// POST /api/v1/pdf/auto-map/:extraction_id
func (h *PDFHandler) RunAutoMapping(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	// Get all items for this extraction
	var items []models.PDFExtractionItem
	if err := h.DB.Where("extraction_id = ? AND mapping_status = ?", extractionID, "pending").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load items"})
		return
	}

	autoMappedCount := 0
	lowConfidenceCount := 0

	// Run auto-mapping for each item
	for _, item := range items {
		// First check for saved package mapping
		var packageMatch *models.ProductPackage
		if h.PackageMapper != nil {
			packageMatch, _ = h.PackageMapper.LookupSavedMapping(item.RawProductText)
		}

		// If package found, use it
		if packageMatch != nil {
			updates := map[string]interface{}{
				"mapped_package_id":  packageMatch.PackageID,
				"mapped_product_id":  nil,
				"mapping_status":     "auto_mapped",
				"mapping_confidence": 100.0,
			}
			if err := h.DB.Model(&models.PDFExtractionItem{}).Where("item_id = ?", item.ItemID).Updates(updates).Error; err != nil {
				log.Printf("warning: failed to update package mapping for item %d: %v", item.ItemID, err)
			} else {
				autoMappedCount++
			}
			continue
		}

		// Otherwise check for product mapping
		suggestion, err := h.Mapper.FindBestMatch(item.RawProductText)
		if err != nil || suggestion == nil {
			continue
		}

		status := "pending"
		if suggestion.Confidence >= 80.0 {
			status = "auto_mapped"
		}

		updates := buildSuggestionUpdates(suggestion, status)
		if updates == nil {
			continue
		}

		if status == "auto_mapped" {
			autoMappedCount++
		} else {
			lowConfidenceCount++
		}

		if err := h.DB.Model(&models.PDFExtractionItem{}).Where("item_id = ?", item.ItemID).Updates(updates).Error; err != nil {
			log.Printf("warning: failed to update mapping for item %d: %v", item.ItemID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"auto_mapped":    autoMappedCount,
		"low_confidence": lowConfidenceCount,
		"message":        fmt.Sprintf("Auto-mapped %d items with high confidence, %d items need manual review", autoMappedCount, lowConfidenceCount),
	})
}

// SearchProducts searches products for manual mapping
// GET /api/v1/pdf/products/search?q=term
func (h *PDFHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter required"})
		return
	}

	var products []models.Product
	searchPattern := "%" + query + "%"

	if err := h.DB.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern).
		Limit(20).
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

// SearchPackages searches WarehouseCore packages for manual mapping
func (h *PDFHandler) SearchPackages(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter required"})
		return
	}

	searchPattern := "%" + query + "%"

	var packages []models.ProductPackage
	if err := h.DB.
		Where("name LIKE ? OR package_code LIKE ? OR description LIKE ?", searchPattern, searchPattern, searchPattern).
		Limit(20).
		Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	results := make([]*packageSummary, 0, len(packages))
	for i := range packages {
		results = append(results, sanitizePackage(&packages[i]))
	}

	c.JSON(http.StatusOK, gin.H{"packages": results})
}

// SearchRentalEquipment searches rental equipment for OCR mapping
// GET /api/v1/pdf/rental-equipment/search?q=term
func (h *PDFHandler) SearchRentalEquipment(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter required"})
		return
	}

	type rentalResult struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Supplier string `json:"supplier"`
	}

	pattern := "%" + query + "%"
	var results []rentalResult
	if err := h.DB.Raw(
		`SELECT id, name, COALESCE(supplier, '') AS supplier
		 FROM rental_equipment
		 WHERE is_active = true AND (name ILIKE ? OR supplier ILIKE ?)
		 ORDER BY name LIMIT 5`,
		pattern, pattern,
	).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rental_equipment": results})
}

// SearchCustomers searches customers for manual mapping
// GET /api/v1/pdf/customers/search?q=term
func (h *PDFHandler) SearchCustomers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter required"})
		return
	}

	pattern := "%" + query + "%"
	var customers []models.Customer
	if err := h.DB.Where("companyname LIKE ? OR lastname LIKE ? OR firstname LIKE ?", pattern, pattern, pattern).
		Limit(20).
		Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Customer search failed"})
		return
	}

	results := make([]gin.H, 0, len(customers))
	for _, customer := range customers {
		results = append(results, gin.H{
			"customerid":   customer.CustomerID,
			"displayName":  customer.GetDisplayName(),
			"companyname":  customer.CompanyName,
			"firstname":    customer.FirstName,
			"lastname":     customer.LastName,
			"city":         customer.City,
			"email":        customer.Email,
			"phonenumber":  customer.PhoneNumber,
			"customertype": customer.CustomerType,
		})
	}

	c.JSON(http.StatusOK, gin.H{"customers": results})
}

// GetDuplicateJobCandidates returns possible duplicate jobs for an extraction
func (h *PDFHandler) GetDuplicateJobCandidates(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"})
		return
	}

	var items []models.PDFExtractionItem
	if err := h.DB.Where("extraction_id = ?", extraction.ExtractionID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load items"})
		return
	}

	productCounts, mappedItems, pendingItems := h.summarizeExtractionItems(items)

	customerID := uint(0)
	if cid := strings.TrimSpace(c.Query("customer_id")); cid != "" {
		if parsed, err := strconv.ParseUint(cid, 10, 32); err == nil && parsed > 0 {
			customerID = uint(parsed)
		}
	}
	if customerID == 0 && extraction.CustomerID.Valid && extraction.CustomerID.Int64 > 0 {
		customerID = uint(extraction.CustomerID.Int64)
	}

	ready := mappedItems > 0 && customerID > 0

	response := gin.H{
		"matches": []duplicateJobMatch{},
		"ready":   ready,
		"pending": pendingItems,
		"checked": false,
	}

	if !ready {
		c.JSON(http.StatusOK, response)
		return
	}

	excludeJobID := uint(0)
	var upload models.PDFUpload
	if err := h.DB.First(&upload, extraction.UploadID).Error; err == nil {
		if upload.JobID.Valid && upload.JobID.Int64 > 0 {
			excludeJobID = uint(upload.JobID.Int64)
		}
	}

	matches, err := h.detectDuplicateJobs(customerID, productCounts, excludeJobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check duplicates"})
		return
	}

	response["matches"] = matches
	response["checked"] = true

	c.JSON(http.StatusOK, response)
}

// SaveManualMapping saves a manual product mapping
// POST /api/v1/pdf/manual-map/:item_id
func (h *PDFHandler) SaveManualMapping(c *gin.Context) {
	itemID := c.Param("item_id")

	var req struct {
		ProductID *int   `json:"product_id"`
		PackageID *int   `json:"package_id"`
		ItemType  string `json:"item_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetPackage := req.PackageID != nil && *req.PackageID > 0
	targetProduct := req.ProductID != nil && *req.ProductID > 0
	if !targetPackage && !targetProduct {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id or package_id is required"})
		return
	}

	var item models.PDFExtractionItem
	if err := h.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	updates := map[string]interface{}{
		"mapping_status":     "user_confirmed",
		"mapping_confidence": 100.0,
	}

	var resultProduct *models.Product
	var resultPackage *packageSummary

	if targetPackage {
		updates["mapped_package_id"] = *req.PackageID
		updates["mapped_product_id"] = nil

		var pkg models.ProductPackage
		if err := h.DB.First(&pkg, *req.PackageID).Error; err == nil {
			resultPackage = sanitizePackage(&pkg)
		}
	} else if targetProduct {
		updates["mapped_product_id"] = *req.ProductID
		updates["mapped_package_id"] = nil

		var product models.Product
		if err := h.DB.First(&product, *req.ProductID).Error; err == nil {
			resultProduct = &product
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping payload"})
		return
	}

	if err := h.DB.Model(&models.PDFExtractionItem{}).Where("item_id = ?", itemID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save mapping"})
		return
	}

	userID := int64(1)
	if uid, exists := c.Get("userid"); exists {
		if id, ok := uid.(int64); ok {
			userID = id
		}
	}

	if targetProduct && req.ProductID != nil {
		if err := h.Mapper.SaveMapping(item.RawProductText, *req.ProductID, userID); err != nil {
			log.Printf("warning: failed to persist manual mapping for item %d: %v", item.ItemID, err)
		}
		h.recordMappingEvent(item.ExtractionID, item.ItemID, *req.ProductID, 0, item.RawProductText, userID)
	} else if targetPackage && req.PackageID != nil && h.PackageMapper != nil {
		if err := h.PackageMapper.SaveMapping(item.RawProductText, *req.PackageID, userID); err != nil {
			log.Printf("warning: failed to persist manual package mapping for item %d: %v", item.ItemID, err)
		}
		h.recordMappingEvent(item.ExtractionID, item.ItemID, 0, *req.PackageID, item.RawProductText, userID)
	}

	response := gin.H{
		"success":    true,
		"item_id":    item.ItemID,
		"confidence": 100.0,
	}

	if resultProduct != nil {
		response["product"] = resultProduct
	}
	if resultPackage != nil {
		response["package"] = resultPackage
	}

	if targetPackage {
		response["message"] = "Package mapping saved successfully"
	} else {
		response["message"] = "Mapping saved and learned for future use"
	}

	c.JSON(http.StatusOK, response)
}

// SaveCustomerMapping saves a manual customer mapping for the extraction
// POST /api/v1/pdf/customer-map/:extraction_id
func (h *PDFHandler) SaveCustomerMapping(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	var req struct {
		CustomerID   int    `json:"customer_id" binding:"required"`
		CustomerText string `json:"customer_text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.CustomerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid customer_id is required"})
		return
	}

	updates := map[string]interface{}{
		"customer_id": req.CustomerID,
	}

	if strings.TrimSpace(req.CustomerText) != "" {
		updates["customer_name"] = req.CustomerText
	}

	if err := h.DB.Model(&models.PDFExtraction{}).Where("extraction_id = ?", extractionID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update extraction"})
		return
	}

	text := strings.TrimSpace(req.CustomerText)
	if text == "" {
		var extraction models.PDFExtraction
		if err := h.DB.Select("customer_name").First(&extraction, extractionID).Error; err == nil && extraction.CustomerName.Valid {
			text = extraction.CustomerName.String
		}
	}

	if text != "" && h.CustomerMapper != nil {
		userID := int64(1)
		if uid, exists := c.Get("userid"); exists {
			if id, ok := uid.(int64); ok {
				userID = id
			}
		}
		_ = h.CustomerMapper.SaveMapping(text, req.CustomerID, userID)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateCustomerFromExtraction creates a CRM customer based on OCR data
func (h *PDFHandler) CreateCustomerFromExtraction(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"})
		return
	}

	var req struct {
		CompanyName string `json:"company_name"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Street      string `json:"street"`
		Zip         string `json:"zip"`
		City        string `json:"city"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	if strings.TrimSpace(req.CompanyName) == "" && strings.TrimSpace(req.LastName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Company or last name is required"})
		return
	}

	customer := &models.Customer{}
	customer.CompanyName = optionalString(req.CompanyName)
	customer.FirstName = optionalString(req.FirstName)
	customer.LastName = optionalString(req.LastName)
	customer.Street = optionalString(req.Street)
	customer.ZIP = optionalString(req.Zip)
	customer.City = optionalString(req.City)

	if err := h.JobHandler.customerRepo.Create(customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	displayName := customer.GetDisplayName()
	updates := map[string]interface{}{
		"customer_id": sql.NullInt64{Int64: int64(customer.CustomerID), Valid: true},
	}
	if displayName != "" {
		updates["customer_name"] = sql.NullString{String: displayName, Valid: true}
	}

	if err := h.DB.Model(&models.PDFExtraction{}).Where("extraction_id = ?", extraction.ExtractionID).Updates(updates).Error; err != nil {
		log.Printf("warning: failed to update extraction with new customer: %v", err)
	}

	if h.CustomerMapper != nil && displayName != "" {
		_ = h.CustomerMapper.SaveMapping(displayName, int(customer.CustomerID), 1)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"customer_id":  customer.CustomerID,
		"display_name": displayName,
	})
}

// FinalizeExtraction creates or links a job for the PDF extraction
// POST /api/v1/pdf/extractions/:extraction_id/finalize
func (h *PDFHandler) FinalizeExtraction(c *gin.Context) {
	extractionID := c.Param("extraction_id")

	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"})
		return
	}

	var upload models.PDFUpload
	if err := h.DB.First(&upload, extraction.UploadID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}

	var req struct {
		StartDate     string   `json:"start_date"`
		EndDate       string   `json:"end_date"`
		CustomerID    *int     `json:"customer_id"`
		DiscountValue *float64 `json:"discount_value"`
		DiscountType  string   `json:"discount_type"`
	}

	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid finalize payload"})
			return
		}
	}

	if req.CustomerID != nil && *req.CustomerID > 0 {
		extraction.CustomerID = sql.NullInt64{Int64: int64(*req.CustomerID), Valid: true}
		h.DB.Model(&models.PDFExtraction{}).Where("extraction_id = ?", extraction.ExtractionID).
			Update("customer_id", extraction.CustomerID)
	}

	if err := h.persistExtractionMappings(c, extraction.ExtractionID); err != nil {
		log.Printf("warning: failed to persist mappings for extraction %d: %v", extraction.ExtractionID, err)
	}

	var meta map[string]string
	if extraction.Metadata.Valid {
		_ = json.Unmarshal([]byte(extraction.Metadata.String), &meta)
	}
	if meta == nil {
		meta = map[string]string{}
	}

	discountType := strings.TrimSpace(meta["discount_type"])
	if discountType != "percent" {
		discountType = "amount"
	}

	if req.DiscountType != "" {
		candidate := strings.ToLower(strings.TrimSpace(req.DiscountType))
		if candidate == "percent" {
			discountType = "percent"
		} else {
			discountType = "amount"
		}
	}

	if req.DiscountValue != nil {
		if *req.DiscountValue > 0 {
			extraction.DiscountAmount = sql.NullFloat64{Float64: *req.DiscountValue, Valid: true}
		} else {
			extraction.DiscountAmount = sql.NullFloat64{}
		}
		if err := h.DB.Model(&models.PDFExtraction{}).
			Where("extraction_id = ?", extraction.ExtractionID).
			Update("discount_amount", extraction.DiscountAmount).Error; err != nil {
			log.Printf("warning: failed to update extraction discount: %v", err)
		}
	}

	meta["discount_type"] = discountType

	discountValue := 0.0
	if extraction.DiscountAmount.Valid {
		discountValue = extraction.DiscountAmount.Float64
	}

	// If job already linked, keep it in sync with the latest mappings
	if upload.JobID.Valid {
		var job models.Job
		if err := h.DB.First(&job, upload.JobID.Int64).Error; err == nil {
			warningMsg, assignErr := h.assignProductsToJob(&job, extraction.ExtractionID)
			if assignErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": assignErr.Error()})
				return
			}

			if err := h.DB.Model(&models.Job{}).Where("job_id = ?", job.JobID).
				Updates(map[string]interface{}{
					"discount":      discountValue,
					"discount_type": discountType,
				}).Error; err != nil {
				log.Printf("warning: failed to persist updated discount for job %d: %v", job.JobID, err)
			}

			_ = h.JobHandler.jobRepo.CalculateAndUpdateRevenue(job.JobID)

			h.attachUploadToJob(&upload, job.JobID)

			response := gin.H{
				"success":  true,
				"job_id":   job.JobID,
				"jobs_url": fmt.Sprintf("/jobs?editJob=%d", job.JobID),
			}
			if warningMsg != "" {
				response["warning"] = warningMsg
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	customerID, err := h.ensureCustomerForExtraction(&extraction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusID, err := h.findDefaultJobStatus()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parseMetaDate := func(value string) *time.Time {
		if value == "" {
			return nil
		}
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			return &t
		}
		return nil
	}

	startDate := parseMetaDate(meta["start_date"])
	endDate := parseMetaDate(meta["end_date"])

	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = &t
			meta["start_date"] = t.Format(time.RFC3339)
		}
	}

	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = &t
			meta["end_date"] = t.Format(time.RFC3339)
		}
	}

	if startDate == nil && extraction.DocumentDate.Valid {
		value := extraction.DocumentDate.Time
		startDate = &value
		meta["start_date"] = value.Format(time.RFC3339)
	}

	if len(meta) > 0 {
		if metaBytes, err := json.Marshal(meta); err == nil {
			h.DB.Model(&models.PDFExtraction{}).Where("extraction_id = ?", extraction.ExtractionID).
				Update("metadata", string(metaBytes))
			extraction.Metadata = sql.NullString{String: string(metaBytes), Valid: true}
		}
	}

	revenue := 0.0
	if extraction.TotalAmount.Valid {
		revenue = extraction.TotalAmount.Float64
	}

	desc := fmt.Sprintf("Generated from %s (Extraction %d)", upload.OriginalFilename, extraction.ExtractionID)
	truncatedDesc := truncateString(desc, 48)

	job := models.Job{
		CustomerID:   customerID,
		StatusID:     statusID,
		Discount:     discountValue,
		DiscountType: discountType,
		Revenue:      revenue,
	}
	job.Description = &truncatedDesc
	if startDate != nil {
		job.StartDate = startDate
	}
	if endDate != nil {
		job.EndDate = endDate
	}

	if err := h.DB.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create job: %v", err),
		})
		return
	}

	warningMsg, assignErr := h.assignProductsToJob(&job, extraction.ExtractionID)
	if assignErr != nil {
		h.DB.Delete(&job)
		c.JSON(http.StatusBadRequest, gin.H{"error": assignErr.Error()})
		return
	}

	h.DB.Model(&models.PDFUpload{}).Where("upload_id = ?", upload.UploadID).
		Update("job_id", job.JobID)

	h.attachUploadToJob(&upload, job.JobID)

	response := gin.H{
		"success":  true,
		"job_id":   job.JobID,
		"jobs_url": fmt.Sprintf("/jobs?editJob=%d", job.JobID),
	}
	if warningMsg != "" {
		response["warning"] = warningMsg
	}

	c.JSON(http.StatusOK, response)
}

func (h *PDFHandler) ensureCustomerForExtraction(extraction *models.PDFExtraction) (uint, error) {
	if extraction.CustomerID.Valid && extraction.CustomerID.Int64 > 0 {
		var existing models.Customer
		if err := h.DB.First(&existing, extraction.CustomerID.Int64).Error; err == nil {
			return existing.CustomerID, nil
		}
	}

	return 0, fmt.Errorf("Please select a customer before creating this job")
}

func (h *PDFHandler) findDefaultJobStatus() (uint, error) {
	var status models.Status
	if err := h.DB.Where("status LIKE ?", "%Draft%").First(&status).Error; err == nil {
		return status.StatusID, nil
	}

	if err := h.DB.Order("statusid").First(&status).Error; err == nil {
		return status.StatusID, nil
	}

	return 0, fmt.Errorf("no job status configured")
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

type productPricingAggregate struct {
	totalAmount    float64
	pricedQuantity int
}

type packageAggregate struct {
	quantity    int
	totalAmount float64
	itemCount   int
	hasPrice    bool
}

func (h *PDFHandler) assignProductsToJob(job *models.Job, extractionID uint64) (string, error) {
	if h.JobHandler == nil {
		return "", nil
	}

	var items []models.PDFExtractionItem
	if err := h.DB.Where("extraction_id = ?", extractionID).Find(&items).Error; err != nil {
		return "", err
	}

	baseProductCounts := make(map[uint]int)
	basePricingAggregates := make(map[uint]*productPricingAggregate)
	packageAggregates := make(map[int]*packageAggregate)
	packageNeeds := make(map[int]map[uint]int)   // package_id -> productID -> qty
	packageUnitPrice := make(map[uint]float64)   // productID -> discounted unit price
	packageComponentTotals := make(map[uint]int) // productID -> qty from packages
	packagePriceContribution := make(map[uint]float64)
	packageQtyContribution := make(map[uint]int)

	// Categorize items: products vs packages
	for _, item := range items {
		switch {
		case item.MappedProductID.Valid:
			pid := uint(item.MappedProductID.Int64)
			qty := getItemQuantity(&item)
			if qty <= 0 {
				continue
			}
			baseProductCounts[pid] += qty
			if lineTotal, hasPrice := resolveLinePricing(&item, qty); hasPrice {
				agg := basePricingAggregates[pid]
				if agg == nil {
					agg = &productPricingAggregate{}
					basePricingAggregates[pid] = agg
				}
				agg.totalAmount += lineTotal
				agg.pricedQuantity += qty
			}

		case item.MappedPackageID.Valid:
			pkgID := int(item.MappedPackageID.Int64)
			qty := getItemQuantity(&item)
			if qty <= 0 {
				continue
			}
			agg := packageAggregates[pkgID]
			if agg == nil {
				agg = &packageAggregate{}
				packageAggregates[pkgID] = agg
			}
			agg.quantity += qty
			agg.itemCount++
			if lineTotal, hasPrice := resolveLinePricing(&item, qty); hasPrice {
				agg.totalAmount += lineTotal
				agg.hasPrice = true
			}
		}
	}

	var warnings []string

	// Expand packages into component counts and discounted pricing
	if len(packageAggregates) > 0 {
		userID := uint(1)
		if job.CreatedBy != nil {
			userID = *job.CreatedBy
		}

		for pkgID, agg := range packageAggregates {
			var pkg models.ProductPackage
			if err := h.DB.Where("package_id = ?", pkgID).First(&pkg).Error; err != nil {
				msg := fmt.Sprintf("Package %d nicht gefunden: %v", pkgID, err)
				log.Printf("Warning: %s", msg)
				warnings = append(warnings, msg)
				continue
			}

			var pkgItems []models.ProductPackageItem
			if err := h.DB.Where("package_id = ?", pkgID).Find(&pkgItems).Error; err != nil {
				msg := fmt.Sprintf("Package %s: Komponenten nicht ladbar (%v)", pkg.Name, err)
				log.Printf("Warning: %s", msg)
				warnings = append(warnings, msg)
				continue
			}

			totalNeededQty := 0
			regularTotal := 0.0
			for _, it := range pkgItems {
				needed := agg.quantity * it.Quantity
				totalNeededQty += needed

				var prod models.Product
				if err := h.DB.First(&prod, it.ProductID).Error; err == nil && prod.ItemCostPerDay != nil && *prod.ItemCostPerDay > 0 {
					regularTotal += *prod.ItemCostPerDay * float64(needed)
				}
			}

			packageTotal := agg.totalAmount
			if packageTotal <= 0 && pkg.Price.Valid {
				packageTotal = pkg.Price.Float64 * float64(agg.quantity)
			}
			if packageTotal < 0 {
				packageTotal = 0
			}
			if packageTotal == 0 && regularTotal > 0 {
				packageTotal = regularTotal
			}

			discountPercent := 0.0
			if regularTotal > 0 {
				discountPercent = 1 - (packageTotal / regularTotal)
				if discountPercent < 0 {
					discountPercent = 0
				}
				if discountPercent > 1 {
					discountPercent = 1
				}
			}

			// Persist job_package metadata (no devices)
			if h.JobPackageRepo != nil {
				perPackage := packageTotal
				if agg.quantity > 0 {
					perPackage = packageTotal / float64(agg.quantity)
				}
				if _, err := h.JobPackageRepo.AssignPackageToJob(int(job.JobID), pkgID, uint(agg.quantity), &perPackage, userID); err != nil {
					log.Printf("Warning: job_package upsert failed for pkg %d job %d: %v", pkgID, job.JobID, err)
					warnings = append(warnings, fmt.Sprintf("Package %s konnte nicht gespeichert werden", pkg.Name))
				}
			}

			for _, it := range pkgItems {
				needed := agg.quantity * it.Quantity
				if needed <= 0 {
					continue
				}

				if packageNeeds[pkgID] == nil {
					packageNeeds[pkgID] = make(map[uint]int)
				}
				pid := uint(it.ProductID)
				packageNeeds[pkgID][pid] += needed
				packageComponentTotals[pid] += needed

				defaultPrice := 0.0
				var prod models.Product
				if err := h.DB.First(&prod, it.ProductID).Error; err == nil && prod.ItemCostPerDay != nil && *prod.ItemCostPerDay > 0 {
					defaultPrice = *prod.ItemCostPerDay
				}

				priceAfterDiscount := 0.0
				if regularTotal > 0 && defaultPrice > 0 {
					priceAfterDiscount = defaultPrice * (1 - discountPercent)
				} else if totalNeededQty > 0 {
					priceAfterDiscount = packageTotal / float64(totalNeededQty)
				}
				if priceAfterDiscount < 0 {
					priceAfterDiscount = 0
				}

				packagePriceContribution[pid] += priceAfterDiscount * float64(needed)
				packageQtyContribution[pid] += needed
			}
		}

		for pid, qty := range packageQtyContribution {
			if qty > 0 {
				unit := packagePriceContribution[pid] / float64(qty)
				if unit < 0 {
					unit = 0
				}
				packageUnitPrice[pid] = unit
			}
		}
	}

	// Combine base + package component counts
	totalCounts := make(map[uint]int)
	for pid, qty := range baseProductCounts {
		totalCounts[pid] = qty
	}
	for pid, qty := range packageComponentTotals {
		totalCounts[pid] += qty
	}

	// Assign devices for total required counts
	if len(totalCounts) > 0 {
		selections := make([]JobProductSelection, 0, len(totalCounts))
		for pid, qty := range totalCounts {
			if qty <= 0 {
				continue
			}
			selections = append(selections, JobProductSelection{
				ProductID: pid,
				Quantity:  qty,
			})
		}

		if len(selections) > 0 {
			if err := h.JobHandler.saveRequirements(job.JobID, selections); err != nil {
				warnings = append(warnings, fmt.Sprintf("Could not save product requirements: %s", err.Error()))
			}
		}
	}

	// Reload devices after assignment
	jobDevices, err := h.JobHandler.jobRepo.GetJobDevices(job.JobID)
	if err != nil {
		return "", err
	}

	// Reset package flags before marking
	h.JobHandler.jobRepo.GetDB().Model(&models.JobDevice{}).
		Where("jobID = ?", job.JobID).
		Updates(map[string]interface{}{
			"is_package_item": false,
			"package_id":      nil,
		})

	// Group devices by product
	devicesByProduct := make(map[uint][]models.JobDevice)
	for _, jd := range jobDevices {
		if jd.Device.ProductID != nil {
			pid := *jd.Device.ProductID
			devicesByProduct[pid] = append(devicesByProduct[pid], jd)
		}
	}

	// Sort package IDs for deterministic assignment
	pkgIDs := make([]int, 0, len(packageNeeds))
	for id := range packageNeeds {
		pkgIDs = append(pkgIDs, id)
	}
	sort.Ints(pkgIDs)

	usedDevices := make(map[string]bool)

	// Assign package items and prices
	for _, pkgID := range pkgIDs {
		needs := packageNeeds[pkgID]
		for pid, need := range needs {
			devices := devicesByProduct[pid]
			count := 0
			for i := 0; i < len(devices) && count < need; i++ {
				jd := devices[i]
				if usedDevices[jd.DeviceID] {
					continue
				}

				updates := map[string]interface{}{
					"is_package_item": true,
					"package_id":      pkgID,
				}
				if price, ok := packageUnitPrice[pid]; ok {
					updates["custom_price"] = price
				}

				h.JobHandler.jobRepo.GetDB().
					Model(&models.JobDevice{}).
					Where("jobID = ? AND deviceID = ?", job.JobID, jd.DeviceID).
					Updates(updates)

				usedDevices[jd.DeviceID] = true
				count++
			}

			if count < need {
				msg := fmt.Sprintf("Package %d: nur %d/%d Geräte für Produkt %d gefunden", pkgID, count, need, pid)
				log.Printf("Warning: %s", msg)
				warnings = append(warnings, msg)
			}
		}
	}

	// Standalone pricing (non-package items)
	standalonePrices := make(map[uint]float64)
	for pid, agg := range basePricingAggregates {
		if agg == nil || agg.pricedQuantity == 0 {
			continue
		}
		unit := agg.totalAmount / float64(agg.pricedQuantity)
		if unit < 0 {
			unit = 0
		}
		standalonePrices[pid] = unit
	}

	for pid, devices := range devicesByProduct {
		price, ok := standalonePrices[pid]
		if !ok {
			continue
		}
		for i := range devices {
			jd := devices[i]
			if usedDevices[jd.DeviceID] {
				continue
			}
			if err := h.JobHandler.jobRepo.UpdateDevicePrice(job.JobID, jd.DeviceID, price); err != nil {
				log.Printf("[WARN] Could not update price for device %s in job %d: %v\n", jd.DeviceID, job.JobID, err)
			}
		}
	}

	// Final revenue update (in case no UpdateDevicePrice was called)
	_ = h.JobHandler.jobRepo.CalculateAndUpdateRevenue(job.JobID)

	// Return combined warnings if any
	if len(warnings) > 0 {
		return strings.Join(warnings, "; "), nil
	}

	return "", nil
}

type customerPrefill struct {
	CompanyName string   `json:"company_name,omitempty"`
	FirstName   string   `json:"first_name,omitempty"`
	LastName    string   `json:"last_name,omitempty"`
	Street      string   `json:"street,omitempty"`
	Zip         string   `json:"zip,omitempty"`
	City        string   `json:"city,omitempty"`
	RawLines    []string `json:"raw_lines,omitempty"`
}

var (
	streetRegex     = regexp.MustCompile(`(?i)^[\p{L}\d\s\.-]+\d+[A-Za-z]?$`)
	zipCityRegex    = regexp.MustCompile(`^(\d{4,5})\s+(.+)$`)
	honorificsRegex = regexp.MustCompile(`(?i)\b(herrn|herr|frau|mr|mrs|ms)\b\.?`)
	addressKeywords = []string{
		"angebotsnr", "angebot nr", "kundennr", "kundennummer", "rechnung", "invoice", "customer no",
	}
	vendorNoise = []string{
		"tsunami events", "ringstraße", "ringstrasse", "haiger", "sparkasse", "steuernummer",
		"amtsgericht", "geschäftsführer", "geschaeftsfuehrer", "iban", "bic", "tel", "telefon",
		"fax", "email", "info@", "www.", "tsunami-events", "tsunami events ug", "haftungsbeschränkt",
	}
)

func (h *PDFHandler) buildCustomerPrefill(extraction *models.PDFExtraction) *customerPrefill {
	if extraction == nil || !extraction.RawText.Valid {
		return nil
	}
	lines := strings.Split(extraction.RawText.String, "\n")
	block := captureRecipientBlock(lines)
	if len(block) == 0 {
		return nil
	}
	cleaned := filterRecipientLines(block)
	if len(cleaned) == 0 {
		return nil
	}

	prefill := &customerPrefill{}
	nameIdx := -1
	for idx, line := range cleaned {
		if containsHonorific(strings.ToLower(line)) {
			nameIdx = idx
			clean := strings.TrimSpace(removeHonorifics(line))
			parts := strings.Fields(clean)
			if len(parts) >= 2 {
				prefill.FirstName = strings.Title(parts[0])
				prefill.LastName = strings.Title(strings.Join(parts[1:], " "))
			} else if len(parts) == 1 {
				prefill.LastName = strings.Title(parts[0])
			}
			break
		}
	}

	recipientLines := cleaned
	if nameIdx >= 0 {
		recipientLines = cleaned[nameIdx:]
	}

	prefill.RawLines = append([]string(nil), recipientLines...)

	if company := extractCompanyLine(cleaned, nameIdx); company != "" {
		prefill.CompanyName = company
	} else if extraction.CustomerName.Valid {
		prefill.CompanyName = extraction.CustomerName.String
	}

	for _, line := range recipientLines {
		if prefill.Street == "" && streetRegex.MatchString(line) {
			prefill.Street = line
			continue
		}
		if prefill.Zip == "" {
			if matches := zipCityRegex.FindStringSubmatch(line); len(matches) == 3 {
				prefill.Zip = matches[1]
				prefill.City = strings.Title(matches[2])
			}
		}
		if nameIdx == -1 && !containsHonorific(strings.ToLower(line)) && prefill.LastName == "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				prefill.FirstName = strings.Title(parts[0])
				prefill.LastName = strings.Title(strings.Join(parts[1:], " "))
				nameIdx = 0
			}
		}
	}

	return prefill
}

func captureRecipientBlock(lines []string) []string {
	if block := captureBlockBeforeKeywords(lines); len(block) > 0 {
		return block
	}
	return captureBlockByHonorific(lines)
}

func captureBlockBeforeKeywords(lines []string) []string {
	var best []string
	bestScore := -1
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if containsAny(lower, addressKeywords) {
			block := []string{}
			collected := 0
			for j := i - 1; j >= 0 && collected < 10; j-- {
				line := strings.TrimSpace(lines[j])
				if line == "" {
					if len(block) > 0 {
						break
					}
					continue
				}
				if isVendorNoise(line) {
					if len(block) > 0 {
						break
					}
					continue
				}
				block = append([]string{line}, block...)
				collected++
			}
			block = trimRecipientBlock(block)
			if len(block) > 0 {
				score := scoreRecipientBlock(block)
				if score > bestScore {
					bestScore = score
					best = block
				}
			}
		}
	}
	return best
}

func captureBlockByHonorific(lines []string) []string {
	best := []string{}
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if containsHonorific(lower) {
			start := i
			for start > 0 {
				prev := strings.TrimSpace(lines[start-1])
				if prev == "" {
					break
				}
				start--
			}
			block := make([]string, 0, 10)
			for j := start; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if t == "" {
					if len(block) > 0 {
						break
					}
					continue
				}
				block = append(block, t)
				if len(block) >= 10 {
					break
				}
			}
			best = trimRecipientBlock(block)
			break
		}
	}
	if len(best) == 0 {
		for _, raw := range lines {
			trimmed := strings.TrimSpace(raw)
			if trimmed != "" && !isVendorNoise(trimmed) {
				best = append(best, trimmed)
			}
			if len(best) >= 6 {
				break
			}
		}
	}
	return best
}

func containsAny(value string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(value, kw) {
			return true
		}
	}
	return false
}

func containsHonorific(value string) bool {
	return strings.Contains(value, "herr") || strings.Contains(value, "frau") || strings.Contains(value, "mr") || strings.Contains(value, "mrs") || strings.Contains(value, "ms")
}

func isVendorNoise(line string) bool {
	lower := strings.ToLower(line)
	for _, kw := range vendorNoise {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func removeHonorifics(value string) string {
	return strings.TrimSpace(honorificsRegex.ReplaceAllString(value, ""))
}

func trimRecipientBlock(lines []string) []string {
	lines = trimLeadingNoise(lines)
	if len(lines) == 0 {
		return lines
	}

	index := -1
	for i, line := range lines {
		if containsHonorific(strings.ToLower(line)) {
			index = i
			break
		}
	}

	if index >= 0 {
		start := index
		if index > 0 {
			prev := strings.TrimSpace(lines[index-1])
			if prev != "" && prev != "," && !isVendorNoise(prev) {
				start = index - 1
			}
		}
		return lines[start:]
	}

	return lines
}

func trimLeadingNoise(lines []string) []string {
	for len(lines) > 0 {
		head := strings.TrimSpace(lines[0])
		if head == "" || head == "," || isVendorNoise(head) {
			lines = lines[1:]
			continue
		}
		break
	}
	return lines
}

func filterRecipientLines(lines []string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isVendorNoise(trimmed) {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return lines
	}
	return result
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func extractCompanyLine(lines []string, honorificIdx int) string {
	if honorificIdx <= 0 {
		return ""
	}
	for i := honorificIdx - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lines[i])
		if candidate == "" || isVendorNoise(candidate) {
			continue
		}
		lower := strings.ToLower(candidate)
		if containsHonorific(lower) {
			continue
		}
		if streetRegex.MatchString(candidate) || zipCityRegex.MatchString(candidate) || containsDigit(candidate) {
			continue
		}
		return candidate
	}
	return ""
}

func scoreRecipientBlock(lines []string) int {
	score := 0
	for _, line := range lines {
		lower := strings.ToLower(line)
		if containsHonorific(lower) {
			score += 4
		}
		if streetRegex.MatchString(line) {
			score += 3
		}
		if zipCityRegex.MatchString(line) {
			score += 3
		}
		if containsDigit(line) {
			score++
		}
		if !isVendorNoise(line) {
			score++
		}
	}
	if len(lines) > 0 {
		score += len(lines)
	}
	return score
}

func containsDigit(value string) bool {
	for _, r := range value {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func (h *PDFHandler) computePriceOverrides(aggregates map[uint]*productPricingAggregate) (map[uint]float64, error) {
	if len(aggregates) == 0 {
		return nil, nil
	}

	productIDs := make([]uint, 0, len(aggregates))
	for pid, agg := range aggregates {
		if agg == nil || agg.pricedQuantity == 0 {
			continue
		}
		productIDs = append(productIDs, pid)
	}

	if len(productIDs) == 0 {
		return nil, nil
	}

	var products []models.Product
	if err := h.DB.Where("productID IN ?", productIDs).Find(&products).Error; err != nil {
		return nil, err
	}

	productMap := make(map[uint]*models.Product, len(products))
	for i := range products {
		product := &products[i]
		productMap[product.ProductID] = product
	}

	overrides := make(map[uint]float64)
	for pid, agg := range aggregates {
		if agg == nil || agg.pricedQuantity == 0 {
			continue
		}
		unitPrice := agg.totalAmount / float64(agg.pricedQuantity)
		if unitPrice < 0 {
			unitPrice = 0
		}

		defaultPrice := 0.0
		if product := productMap[pid]; product != nil && product.ItemCostPerDay != nil {
			defaultPrice = *product.ItemCostPerDay
		}

		if math.Abs(defaultPrice-unitPrice) < 0.01 {
			continue
		}

		overrides[pid] = unitPrice
	}

	if len(overrides) == 0 {
		return nil, nil
	}

	return overrides, nil
}

func (h *PDFHandler) applyCustomPriceOverrides(job *models.Job, overrides map[uint]float64) error {
	if len(overrides) == 0 {
		return nil
	}

	jobDevices, err := h.JobHandler.jobRepo.GetJobDevices(job.JobID)
	if err != nil {
		// Log warning but don't fail - job creation should still succeed
		fmt.Printf("[WARN] Could not fetch job devices for price overrides: %v\n", err)
		return nil
	}

	failedUpdates := 0
	for _, jd := range jobDevices {
		if jd.Device.DeviceID == "" {
			continue
		}
		var productID uint
		if jd.Device.ProductID != nil {
			productID = *jd.Device.ProductID
		} else if jd.Device.Product != nil {
			productID = jd.Device.Product.ProductID
		}
		if productID == 0 {
			continue
		}
		price, ok := overrides[productID]
		if !ok {
			continue
		}
		if price < 0 {
			price = 0
		}
		if err := h.JobHandler.jobRepo.UpdateDevicePrice(job.JobID, jd.DeviceID, price); err != nil {
			// Log warning but continue - don't fail entire job creation
			fmt.Printf("[WARN] Could not update price for device %s in job %d: %v\n", jd.DeviceID, job.JobID, err)
			failedUpdates++
			continue
		}
	}

	if failedUpdates > 0 {
		fmt.Printf("[INFO] %d device price updates failed for job %d, but job was created successfully\n", failedUpdates, job.JobID)
	}

	return nil
}

func (h *PDFHandler) summarizeExtractionItems(items []models.PDFExtractionItem) (map[uint]int, int, int) {
	counts := make(map[uint]int)
	mapped := 0
	unmapped := 0
	packageQuantities := make(map[int]int)

	for _, item := range items {
		switch {
		case item.MappedProductID.Valid:
			mapped++
			qty := getItemQuantity(&item)
			counts[uint(item.MappedProductID.Int64)] += qty
		case item.MappedPackageID.Valid:
			mapped++
			qty := getItemQuantity(&item)
			if qty > 0 {
				packageQuantities[int(item.MappedPackageID.Int64)] += qty
			}
		default:
			unmapped++
		}
	}

	if len(packageQuantities) > 0 {
		h.expandPackageProductCounts(packageQuantities, counts)
	}

	return counts, mapped, unmapped
}

func calculateExtractionItemDiscount(item *models.PDFExtractionItem) float64 {
	if item == nil || !item.UnitPrice.Valid || item.UnitPrice.Float64 <= 0 {
		return 0
	}

	qty := 1
	if item.Quantity.Valid && item.Quantity.Int64 > 0 {
		qty = int(item.Quantity.Int64)
	}

	lineTotal := 0.0
	if item.LineTotal.Valid {
		lineTotal = item.LineTotal.Float64
	}

	expected := item.UnitPrice.Float64 * float64(qty)
	if expected <= 0 {
		return 0
	}

	discount := expected - lineTotal
	if discount <= 0.005 {
		return 0
	}

	return discount
}

func (h *PDFHandler) expandPackageProductCounts(packageQuantities map[int]int, counts map[uint]int) {
	if len(packageQuantities) == 0 {
		return
	}

	ids := make([]int, 0, len(packageQuantities))
	for id := range packageQuantities {
		ids = append(ids, id)
	}

	var packageItems []models.ProductPackageItem
	if err := h.DB.Where("package_id IN ?", ids).Find(&packageItems).Error; err != nil {
		log.Printf("warning: failed to fetch package items: %v", err)
		return
	}

	itemsByPackage := make(map[int][]models.ProductPackageItem)
	for _, pkgItem := range packageItems {
		itemsByPackage[pkgItem.PackageID] = append(itemsByPackage[pkgItem.PackageID], pkgItem)
	}

	for packageID, packageQty := range packageQuantities {
		if packageQty <= 0 {
			continue
		}
		components := itemsByPackage[packageID]
		for _, component := range components {
			if component.ProductID <= 0 || component.Quantity <= 0 {
				continue
			}
			counts[uint(component.ProductID)] += packageQty * component.Quantity
		}
	}
}

func (h *PDFHandler) fallbackAssignPackageComponents(pkg *models.ProductPackage, pkgItems []models.ProductPackageItem, agg *packageAggregate, productCounts map[uint]int, pricingAggregates map[uint]*productPricingAggregate, warnings *[]string) {
	if pkg == nil || agg == nil || agg.quantity <= 0 {
		return
	}
	if len(pkgItems) == 0 {
		if warnings != nil {
			*warnings = append(*warnings, fmt.Sprintf("Package %d besitzt keine Komponenten, kann nicht auf Geräte abbilden", pkg.PackageID))
		}
		return
	}

	// Load involved products to get default pricing
	productIDs := make([]int, 0, len(pkgItems))
	for _, item := range pkgItems {
		productIDs = append(productIDs, item.ProductID)
	}
	var products []models.Product
	if err := h.DB.Where("productID IN ?", productIDs).Find(&products).Error; err != nil {
		if warnings != nil {
			*warnings = append(*warnings, fmt.Sprintf("Package %s: Produkte konnten nicht geladen werden (%v)", pkg.Name, err))
		}
		return
	}
	productMap := make(map[int]*models.Product, len(products))
	for i := range products {
		product := &products[i]
		productMap[int(product.ProductID)] = product
	}

	// Calculate discount based on package price vs. regular component sum
	regularTotal := 0.0
	for _, item := range pkgItems {
		if prod := productMap[item.ProductID]; prod != nil && prod.ItemCostPerDay != nil && *prod.ItemCostPerDay > 0 {
			regularTotal += *prod.ItemCostPerDay * float64(item.Quantity)
		}
	}
	packagePrice := 0.0
	if agg.hasPrice && agg.quantity > 0 {
		packagePrice = agg.totalAmount / float64(agg.quantity)
	} else if pkg.Price.Valid {
		packagePrice = pkg.Price.Float64
	}
	if packagePrice < 0 {
		packagePrice = 0
	}

	discountPercent := 0.0
	totalNeededQty := 0
	if len(pkgItems) > 0 {
		for _, item := range pkgItems {
			totalNeededQty += agg.quantity * item.Quantity
		}
	}
	if regularTotal > 0 {
		discountPercent = (regularTotal - packagePrice) / regularTotal
		if discountPercent < 0 {
			discountPercent = 0
		}
		if discountPercent > 1 {
			discountPercent = 1
		}
	}

	for _, item := range pkgItems {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			continue
		}
		neededQty := agg.quantity * item.Quantity
		if neededQty <= 0 {
			continue
		}

		productCounts[uint(item.ProductID)] += neededQty

		// Apply discounted pricing into aggregates so price overrides mirror the package discount
		prod := productMap[item.ProductID]
		defaultPrice := 0.0
		if prod != nil && prod.ItemCostPerDay != nil && *prod.ItemCostPerDay > 0 {
			defaultPrice = *prod.ItemCostPerDay
		}

		var priceAfterDiscount float64
		if regularTotal > 0 && defaultPrice > 0 {
			priceAfterDiscount = defaultPrice * (1 - discountPercent)
		} else if totalNeededQty > 0 {
			// Evenly distribute when no default prices exist
			priceAfterDiscount = packagePrice / float64(totalNeededQty)
		}
		if priceAfterDiscount < 0 {
			priceAfterDiscount = 0
		}

		aggRec := pricingAggregates[uint(item.ProductID)]
		if aggRec == nil {
			aggRec = &productPricingAggregate{}
			pricingAggregates[uint(item.ProductID)] = aggRec
		}
		aggRec.totalAmount += priceAfterDiscount * float64(neededQty)
		aggRec.pricedQuantity += neededQty
	}
}

func (h *PDFHandler) detectDuplicateJobs(customerID uint, productCounts map[uint]int, excludeJobID uint) ([]duplicateJobMatch, error) {
	if customerID == 0 || len(productCounts) == 0 {
		return nil, nil
	}

	totalDevices := 0
	for _, qty := range productCounts {
		totalDevices += qty
	}
	if totalDevices == 0 {
		return nil, nil
	}

	candidateQuery := h.DB.Table("jobs").
		Select("jobs.jobID").
		Joins("JOIN job_devices jd ON jd.jobID = jobs.jobID").
		Where("jobs.customerID = ?", customerID)
	if excludeJobID > 0 {
		candidateQuery = candidateQuery.Where("jobs.jobID <> ?", excludeJobID)
	}

	var candidateIDs []uint
	if err := candidateQuery.Group("jobs.jobID").
		Having("COUNT(*) = ?", totalDevices).
		Scan(&candidateIDs).Error; err != nil {
		return nil, err
	}
	if len(candidateIDs) == 0 {
		return nil, nil
	}

	type jobProductRow struct {
		JobID     uint
		ProductID uint
		Quantity  int
	}

	var rows []jobProductRow
	if err := h.DB.Table("job_devices AS jd").
		Select("jd.jobID, dev.productID, COUNT(*) AS quantity").
		Joins("JOIN devices dev ON dev.deviceID = jd.deviceID").
		Where("jd.jobID IN ?", candidateIDs).
		Where("dev.productID IS NOT NULL").
		Group("jd.jobID, dev.productID").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	jobCounts := make(map[uint]map[uint]int)
	jobTotals := make(map[uint]int)

	for _, row := range rows {
		if row.ProductID == 0 {
			continue
		}
		if jobCounts[row.JobID] == nil {
			jobCounts[row.JobID] = make(map[uint]int)
		}
		jobCounts[row.JobID][row.ProductID] = row.Quantity
		jobTotals[row.JobID] += row.Quantity
	}

	matchingIDs := make([]uint, 0, len(candidateIDs))
	for _, candidateID := range candidateIDs {
		counts := jobCounts[candidateID]
		if len(counts) == 0 {
			continue
		}
		if len(counts) != len(productCounts) {
			continue
		}
		match := true
		for pid, qty := range productCounts {
			if counts[pid] != qty {
				match = false
				break
			}
		}
		if match {
			matchingIDs = append(matchingIDs, candidateID)
		}
	}

	if len(matchingIDs) == 0 {
		return nil, nil
	}

	sort.Slice(matchingIDs, func(i, j int) bool { return matchingIDs[i] < matchingIDs[j] })

	type jobInfo struct {
		JobID       uint
		JobCode     string
		Description sql.NullString
		StartDate   sql.NullTime
		EndDate     sql.NullTime
		StatusName  sql.NullString
	}

	var infoRows []jobInfo
	if err := h.DB.Table("jobs").
		Select("jobs.jobID, jobs.job_code, jobs.description, jobs.startDate, jobs.endDate, status.status AS status_name").
		Joins("LEFT JOIN status ON status.statusID = jobs.statusID").
		Where("jobs.jobID IN ?", matchingIDs).
		Scan(&infoRows).Error; err != nil {
		return nil, err
	}

	infoMap := make(map[uint]jobInfo, len(infoRows))
	for _, info := range infoRows {
		infoMap[info.JobID] = info
		if jobTotals[info.JobID] == 0 {
			jobTotals[info.JobID] = totalDevices
		}
	}

	results := make([]duplicateJobMatch, 0, len(matchingIDs))
	for _, jobID := range matchingIDs {
		info, ok := infoMap[jobID]
		if !ok {
			continue
		}

		results = append(results, duplicateJobMatch{
			JobID:       jobID,
			JobCode:     info.JobCode,
			Description: info.Description.String,
			StartDate:   formatJobDate(info.StartDate),
			EndDate:     formatJobDate(info.EndDate),
			Status:      info.StatusName.String,
			DeviceCount: jobTotals[jobID],
			JobsURL:     fmt.Sprintf("/jobs?editJob=%d", jobID),
		})
	}

	return results, nil
}

func formatJobDate(value sql.NullTime) string {
	if value.Valid {
		return value.Time.Format("2006-01-02")
	}
	return ""
}

func (h *PDFHandler) attachUploadToJob(upload *models.PDFUpload, jobID uint) {
	if upload == nil || jobID == 0 {
		return
	}

	// New approach: Use DocumentHandler to assign document to job (File Pool as single source of truth)
	if h.DocumentHandler != nil && upload.DocumentID.Valid && upload.DocumentID.Int64 > 0 {
		// Document already exists in File Pool, just update its assignment
		if err := h.DocumentHandler.AssignDocumentToJob(uint(upload.DocumentID.Int64), jobID); err != nil {
			log.Printf("warning: failed to assign document %d to job %d: %v", upload.DocumentID.Int64, jobID, err)
		} else {
			log.Printf("Document %d assigned to job %d via File Pool", upload.DocumentID.Int64, jobID)
			return
		}
	}

	// Fallback: Legacy approach for uploads without document_id (backward compatibility)
	if h.AttachmentRepo == nil {
		return
	}

	sourcePath := strings.TrimSpace(upload.FilePath)
	if sourcePath == "" {
		return
	}

	if _, err := os.Stat(sourcePath); err != nil {
		log.Printf("warning: cannot attach OCR upload %d to job %d: %v", upload.UploadID, jobID, err)
		return
	}

	if err := os.MkdirAll(h.attachmentDir, 0755); err != nil {
		log.Printf("warning: cannot create attachment directory %s: %v", h.attachmentDir, err)
		return
	}

	destFilename := h.buildAttachmentFilename(upload, jobID)
	if existing, err := h.AttachmentRepo.GetByFilename(destFilename); err == nil && existing != nil {
		if existing.JobID == jobID {
			return
		}
	}

	destPath := filepath.Join(h.attachmentDir, destFilename)
	if err := copyFile(sourcePath, destPath); err != nil {
		log.Printf("warning: failed to copy OCR upload %d for job %d: %v", upload.UploadID, jobID, err)
		return
	}

	info, err := os.Stat(destPath)
	if err != nil {
		log.Printf("warning: failed to stat attachment copy for job %d: %v", jobID, err)
		return
	}

	ext := strings.ToLower(filepath.Ext(upload.OriginalFilename))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/pdf"
	}

	attachment := &models.JobAttachment{
		JobID:            jobID,
		Filename:         destFilename,
		OriginalFilename: upload.OriginalFilename,
		FilePath:         destPath,
		FileSize:         info.Size(),
		MimeType:         mimeType,
		Description:      fmt.Sprintf("Imported from OCR upload #%d", upload.UploadID),
		UploadedAt:       time.Now(),
		IsActive:         true,
	}

	if uploader := convertNullInt64ToUintPtr(upload.UploadedBy); uploader != nil {
		attachment.UploadedBy = uploader
	}

	if err := h.AttachmentRepo.Create(attachment); err != nil {
		log.Printf("warning: failed to create attachment record for job %d: %v", jobID, err)
		_ = os.Remove(destPath)
	}
}

func (h *PDFHandler) buildAttachmentFilename(upload *models.PDFUpload, jobID uint) string {
	base := strings.TrimSuffix(upload.OriginalFilename, filepath.Ext(upload.OriginalFilename))
	safeBase := sanitizeFilenameBase(base)
	if safeBase == "" {
		safeBase = "document"
	}
	return fmt.Sprintf("job%d_upload%d_%s%s", jobID, upload.UploadID, safeBase, filepath.Ext(upload.OriginalFilename))
}

var filenameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilenameBase(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sanitized := filenameSanitizer.ReplaceAllString(trimmed, "_")
	return strings.Trim(sanitized, "_")
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}

	return target.Sync()
}

func convertNullInt64ToUintPtr(value sql.NullInt64) *uint {
	if !value.Valid || value.Int64 <= 0 {
		return nil
	}
	u := uint(value.Int64)
	return &u
}

func formatDateInput(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
func (h *PDFHandler) persistExtractionMappings(c *gin.Context, extractionID uint64) error {
	var mappedItems []models.PDFExtractionItem
	if err := h.DB.Where("extraction_id = ? AND (mapped_product_id IS NOT NULL OR mapped_package_id IS NOT NULL)", extractionID).
		Find(&mappedItems).Error; err != nil {
		return err
	}

	if len(mappedItems) == 0 {
		return nil
	}

	userID := int64(0)
	if uid, exists := c.Get("userid"); exists {
		if id, ok := uid.(int64); ok {
			userID = id
		}
	}

	for _, item := range mappedItems {
		text := strings.TrimSpace(item.RawProductText)
		if text == "" {
			continue
		}

		// Handle product mappings
		if item.MappedProductID.Valid {
			if err := h.Mapper.SaveMapping(text, int(item.MappedProductID.Int64), userID); err != nil {
				log.Printf("warning: failed to save product mapping for extraction item %d: %v", item.ItemID, err)
			} else {
				h.recordMappingEvent(item.ExtractionID, item.ItemID, int(item.MappedProductID.Int64), 0, text, userID)
			}
		}

		// Handle package mappings
		if item.MappedPackageID.Valid && h.PackageMapper != nil {
			if err := h.PackageMapper.SaveMapping(text, int(item.MappedPackageID.Int64), userID); err != nil {
				log.Printf("warning: failed to save package mapping for extraction item %d: %v", item.ItemID, err)
			} else {
				h.recordMappingEvent(item.ExtractionID, item.ItemID, 0, int(item.MappedPackageID.Int64), text, userID)
			}
		}
	}

	return nil
}

func (h *PDFHandler) recordMappingEvent(extractionID uint64, itemID uint64, productID int, packageID int, rawText string, userID int64) {
	event := models.PDFMappingEvent{
		PDFProductText: rawText,
	}

	if productID > 0 {
		event.ProductID = sql.NullInt64{Int64: int64(productID), Valid: true}
	}
	if packageID > 0 {
		event.PackageID = sql.NullInt64{Int64: int64(packageID), Valid: true}
	}
	if extractionID > 0 {
		event.ExtractionID = sql.NullInt64{Int64: int64(extractionID), Valid: true}
	}
	if itemID > 0 {
		event.ItemID = sql.NullInt64{Int64: int64(itemID), Valid: true}
	}

	normalized := strings.TrimSpace(strings.ToLower(rawText))
	if normalized != "" {
		event.NormalizedText = sql.NullString{String: normalized, Valid: true}
	}
	if userID > 0 {
		event.CreatedBy = sql.NullInt64{Int64: userID, Valid: true}
	}

	if err := h.DB.Create(&event).Error; err != nil {
		log.Printf("warning: failed to record mapping event: %v", err)
	}
}

// ProcessPoolDocument starts OCR processing from an existing File Pool document
// POST /api/v1/pdf/from-pool/:documentID
func (h *PDFHandler) ProcessPoolDocument(c *gin.Context) {
	documentIDStr := c.Param("documentID")
	documentID, err := strconv.ParseUint(documentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	if h.DocumentHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Document handler not configured"})
		return
	}

	// Get the document from File Pool
	document, err := h.DocumentHandler.GetDocumentByID(uint(documentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found in File Pool"})
		return
	}

	// Validate it's a PDF
	if document.MimeType != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF documents can be processed for OCR"})
		return
	}

	// Get the file path
	var filePath string
	if strings.HasPrefix(document.FilePath, "nextcloud:") {
		// For Nextcloud files, we need to download to a temp location first
		tempDir := filepath.Join(h.Extractor.GetUploadDir(), "temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
			return
		}

		tempPath := filepath.Join(tempDir, document.Filename)
		ncPath := strings.TrimPrefix(document.FilePath, "nextcloud:")

		if h.DocumentHandler.UseNextcloud() {
			reader, _, err := h.DocumentHandler.GetNextcloudClient().Download(ncPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file from storage"})
				return
			}
			defer reader.Close()

			outFile, err := os.Create(tempPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp file"})
				return
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, reader); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save temp file"})
				return
			}
			filePath = tempPath
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Nextcloud not configured"})
			return
		}
	} else {
		filePath = document.FilePath
	}

	// Verify file exists
	info, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document file not found on disk"})
		return
	}

	// Get user ID from session
	var uploadedBy sql.NullInt64
	if userID, exists := c.Get("userid"); exists {
		if uid, ok := userID.(int64); ok {
			uploadedBy = sql.NullInt64{Int64: uid, Valid: true}
		}
	}

	// Create pdf_upload record linked to the document
	upload := &models.PDFUpload{
		DocumentID:       sql.NullInt64{Int64: int64(document.DocumentID), Valid: true},
		OriginalFilename: document.OriginalFilename,
		StoredFilename:   document.Filename,
		FilePath:         filePath,
		FileSize:         info.Size(),
		MimeType:         document.MimeType,
		UploadedBy:       uploadedBy,
		UploadedAt:       time.Now(),
		ProcessingStatus: "pending",
		IsActive:         true,
	}

	if err := h.DB.Create(upload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload record"})
		return
	}

	// Start processing asynchronously
	go h.processUploadAsync(upload.UploadID)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"upload_id":   upload.UploadID,
		"document_id": document.DocumentID,
		"message":     "PDF processing started from File Pool document",
	})
}

// GetPoolDocumentsForOCR returns unassigned PDF documents from the File Pool
// that can be selected for OCR processing
// GET /api/v1/pdf/pool-documents
func (h *PDFHandler) GetPoolDocumentsForOCR(c *gin.Context) {
	if h.DocumentHandler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Document handler not configured"})
		return
	}

	// Auto-sync from Nextcloud before listing
	h.DocumentHandler.SyncFromNextcloud()

	// Get unassigned PDF documents
	var documents []models.Document
	if err := h.DocumentHandler.GetDB().
		Where("entity_type = ? AND entity_id = ? AND mime_type = ?", "system", "unassigned", "application/pdf").
		Order("uploaded_at DESC").
		Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load documents"})
		return
	}

	// Format response
	type docResponse struct {
		DocumentID       uint   `json:"document_id"`
		Filename         string `json:"filename"`
		OriginalFilename string `json:"original_filename"`
		FileSize         int64  `json:"file_size"`
		UploadedAt       string `json:"uploaded_at"`
	}

	results := make([]docResponse, 0, len(documents))
	for _, doc := range documents {
		results = append(results, docResponse{
			DocumentID:       doc.DocumentID,
			Filename:         doc.Filename,
			OriginalFilename: doc.OriginalFilename,
			FileSize:         doc.FileSize,
			UploadedAt:       doc.UploadedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"documents": results,
		"count":     len(results),
	})
}

// MappingRow is the enriched view model for the mapping management page and API.
type MappingRow struct {
	MappingID       uint64  `json:"mapping_id"`
	OCRText         string  `json:"ocr_text"`
	TargetName      string  `json:"target_name"`
	TargetType      string  `json:"target_type"` // "product", "package", or "customer"
	TargetID        int     `json:"target_id"`
	MappingType     string  `json:"mapping_type"`
	ConfidenceScore float64 `json:"confidence_score"`
	UsageCount      int     `json:"usage_count"`
}

// buildAllMappingRows combines product, package, and customer mappings into enriched MappingRows.
func buildAllMappingRows(
	productMappings []models.PDFProductMapping,
	packageMappings []models.PDFPackageMapping,
	customerMappings []models.PDFCustomerMapping,
	productNames map[int]string,
	packageNames map[int]string,
	customerNames map[int]string,
) []MappingRow {
	rows := make([]MappingRow, 0, len(productMappings)+len(packageMappings)+len(customerMappings))
	for _, m := range productMappings {
		name, ok := productNames[m.ProductID]
		if !ok { name = fmt.Sprintf("Product #%d", m.ProductID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFProductText, TargetName: name,
			TargetType: "product", TargetID: m.ProductID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	for _, m := range packageMappings {
		name, ok := packageNames[m.PackageID]
		if !ok { name = fmt.Sprintf("Package #%d", m.PackageID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFPackageText, TargetName: name,
			TargetType: "package", TargetID: m.PackageID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	for _, m := range customerMappings {
		name, ok := customerNames[m.CustomerID]
		if !ok { name = fmt.Sprintf("Customer #%d", m.CustomerID) }
		conf := 0.0
		if m.ConfidenceScore.Valid { conf = m.ConfidenceScore.Float64 }
		rows = append(rows, MappingRow{
			MappingID: m.MappingID, OCRText: m.PDFCustomerText, TargetName: name,
			TargetType: "customer", TargetID: m.CustomerID, MappingType: m.MappingType,
			ConfidenceScore: conf, UsageCount: m.UsageCount,
		})
	}
	return rows
}

// GetAllMappingsAPI returns active mappings as JSON.
// GET /api/v1/pdf/mappings?type=product|package|customer  (omit for all)
func (h *PDFHandler) GetAllMappingsAPI(c *gin.Context) {
	typeFilter := strings.ToLower(c.DefaultQuery("type", ""))

	var productMappings []models.PDFProductMapping
	var packageMappings []models.PDFPackageMapping
	var customerMappings []models.PDFCustomerMapping
	var err error

	if typeFilter == "" || typeFilter == "product" {
		productMappings, err = h.Mapper.GetAllMappings()
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load product mappings"}); return }
	}
	if typeFilter == "" || typeFilter == "package" {
		if h.PackageMapper != nil {
			packageMappings, err = h.PackageMapper.GetAllMappings()
			if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load package mappings"}); return }
		}
	}
	if typeFilter == "" || typeFilter == "customer" {
		if h.CustomerMapper != nil {
			if err := h.DB.Where("is_active = true").Order("usage_count DESC").Find(&customerMappings).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer mappings"}); return
			}
		}
	}

	productNames := make(map[int]string)
	if len(productMappings) > 0 {
		ids := make([]int, 0, len(productMappings))
		for _, m := range productMappings { ids = append(ids, m.ProductID) }
		var products []models.Product
		if err := h.DB.Select("productid, name").Where("productid IN ?", ids).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load product names"}); return
		}
		for _, p := range products { productNames[int(p.ProductID)] = p.Name }
	}

	packageNames := make(map[int]string)
	if len(packageMappings) > 0 {
		ids := make([]int, 0, len(packageMappings))
		for _, m := range packageMappings { ids = append(ids, m.PackageID) }
		var packages []models.ProductPackage
		if err := h.DB.Select("package_id, name").Where("package_id IN ?", ids).Find(&packages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load package names"}); return
		}
		for _, p := range packages { packageNames[p.PackageID] = p.Name }
	}

	customerNames := make(map[int]string)
	if len(customerMappings) > 0 {
		ids := make([]int, 0, len(customerMappings))
		for _, m := range customerMappings { ids = append(ids, m.CustomerID) }
		var customers []models.Customer
		if err := h.DB.Select("customerid, firstname, lastname, companyname").Where("customerid IN ?", ids).Find(&customers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customer names"}); return
		}
		for _, cu := range customers {
			customerNames[int(cu.CustomerID)] = cu.GetDisplayName()
		}
	}

	rows := buildAllMappingRows(productMappings, packageMappings, customerMappings, productNames, packageNames, customerNames)
	c.JSON(http.StatusOK, gin.H{"mappings": rows})
}

// DeleteMappingAPI soft-deletes a mapping (sets is_active = false).
// DELETE /api/v1/pdf/mappings/:id?type=product|package
func (h *PDFHandler) DeleteMappingAPI(c *gin.Context) {
	idParam := c.Param("id")
	mappingID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}

	mappingType := strings.ToLower(c.DefaultQuery("type", "product"))

	switch mappingType {
	case "product":
		if err := h.Mapper.DeleteMapping(mappingID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mapping"})
			return
		}
	case "package":
		if h.PackageMapper == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Package mapper not available"})
			return
		}
		if err := h.PackageMapper.DeleteMapping(mappingID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mapping"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'product' or 'package'"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateMappingAPI changes the target product or package for an existing mapping.
// PUT /api/v1/pdf/mappings/:id
// Body: { "type": "product"|"package", "target_id": 15 }
func (h *PDFHandler) UpdateMappingAPI(c *gin.Context) {
	idParam := c.Param("id")
	mappingID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}

	var req struct {
		Type     string `json:"type" binding:"required"`
		TargetID int    `json:"target_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and target_id are required"})
		return
	}
	if req.TargetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_id must be a positive integer"})
		return
	}

	switch strings.ToLower(req.Type) {
	case "product":
		var product models.Product
		if err := h.DB.First(&product, req.TargetID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		result := h.DB.Model(&models.PDFProductMapping{}).
			Where("mapping_id = ?", mappingID).
			Updates(map[string]interface{}{
				"product_id":   req.TargetID,
				"mapping_type": "manual",
				"is_active":    true,
			})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mapping"})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "target_name": product.Name})

	case "package":
		if h.PackageMapper == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Package mapper not available"})
			return
		}
		var pkg models.ProductPackage
		if err := h.DB.First(&pkg, req.TargetID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
			return
		}
		result := h.DB.Model(&models.PDFPackageMapping{}).
			Where("mapping_id = ?", mappingID).
			Updates(map[string]interface{}{
				"package_id":   req.TargetID,
				"mapping_type": "manual",
				"is_active":    true,
			})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mapping"})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "target_name": pkg.Name})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'product' or 'package'"})
	}
}

// ShowMappingManagement renders the global mapping management page.
// GET /settings/mappings
func (h *PDFHandler) ShowMappingManagement(c *gin.Context) {
	productMappings, err := h.Mapper.GetAllMappings()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	var packageMappings []models.PDFPackageMapping
	if h.PackageMapper != nil {
		packageMappings, err = h.PackageMapper.GetAllMappings()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
	}

	productIDSet := make(map[int]struct{})
	for _, m := range productMappings {
		productIDSet[m.ProductID] = struct{}{}
	}
	productNames := make(map[int]string)
	if len(productIDSet) > 0 {
		ids := make([]int, 0, len(productIDSet))
		for id := range productIDSet {
			ids = append(ids, id)
		}
		var products []models.Product
		if err := h.DB.Select("productid, name").Where("productid IN ?", ids).Find(&products).Error; err != nil {
			c.String(http.StatusInternalServerError, "Failed to load mapping data")
			return
		}
		for _, p := range products {
			productNames[int(p.ProductID)] = p.Name
		}
	}

	packageIDSet := make(map[int]struct{})
	for _, m := range packageMappings {
		packageIDSet[m.PackageID] = struct{}{}
	}
	packageNames := make(map[int]string)
	if len(packageIDSet) > 0 {
		ids := make([]int, 0, len(packageIDSet))
		for id := range packageIDSet {
			ids = append(ids, id)
		}
		var packages []models.ProductPackage
		if err := h.DB.Select("package_id, name").Where("package_id IN ?", ids).Find(&packages).Error; err != nil {
			c.String(http.StatusInternalServerError, "Failed to load mapping data")
			return
		}
		for _, p := range packages {
			packageNames[p.PackageID] = p.Name
		}
	}

	var customerMappings []models.PDFCustomerMapping
	if h.CustomerMapper != nil {
		if err := h.DB.Where("is_active = true").Order("usage_count DESC").Find(&customerMappings).Error; err != nil {
			log.Printf("warning: failed to load customer mappings for management page: %v", err)
		}
	}
	customerIDSet := make(map[int]struct{})
	for _, m := range customerMappings { customerIDSet[m.CustomerID] = struct{}{} }
	customerNames := make(map[int]string)
	if len(customerIDSet) > 0 {
		cids := make([]int, 0, len(customerIDSet))
		for id := range customerIDSet { cids = append(cids, id) }
		var customers []models.Customer
		if err := h.DB.Select("customerid, firstname, lastname, companyname").Where("customerid IN ?", cids).Find(&customers).Error; err == nil {
			for _, cu := range customers {
				customerNames[int(cu.CustomerID)] = cu.GetDisplayName()
			}
		}
	}
	rows := buildAllMappingRows(productMappings, packageMappings, customerMappings, productNames, packageNames, customerNames)

	user, _ := c.Get("user")
	c.HTML(http.StatusOK, "mapping_management.html", gin.H{
		"mappings":    rows,
		"title":       "Produkt-Mappings",
		"currentPage": "mappings",
		"user":        user,
	})
}

// GetExtractionPreview returns mapped items with resolved product names for preview step.
// GET /api/v1/pdf/extractions/:extraction_id/preview
func (h *PDFHandler) GetExtractionPreview(c *gin.Context) {
	extractionID := c.Param("extraction_id")
	var extraction models.PDFExtraction
	if err := h.DB.First(&extraction, extractionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extraction not found"})
		return
	}
	var items []models.PDFExtractionItem
	if err := h.DB.Where("extraction_id = ? AND (mapped_product_id IS NOT NULL OR mapped_package_id IS NOT NULL)", extractionID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load extraction items"})
		return
	}

	type PreviewItem struct {
		ItemID     uint64  `json:"item_id"`
		Name       string  `json:"name"`
		RawText    string  `json:"raw_text"`
		Quantity   int     `json:"quantity"`
		UnitPrice  float64 `json:"unit_price"`
		LineTotal  float64 `json:"line_total"`
		TargetType string  `json:"target_type"`
		TargetID   int     `json:"target_id"`
	}

	productIDs, packageIDs := []int{}, []int{}
	for _, it := range items {
		if it.MappedProductID.Valid { productIDs = append(productIDs, int(it.MappedProductID.Int64)) }
		if it.MappedPackageID.Valid { packageIDs = append(packageIDs, int(it.MappedPackageID.Int64)) }
	}
	productNames := make(map[int]string)
	if len(productIDs) > 0 {
		var products []models.Product
		h.DB.Select("productid, name").Where("productid IN ?", productIDs).Find(&products)
		for _, p := range products { productNames[int(p.ProductID)] = p.Name }
	}
	packageNames := make(map[int]string)
	if len(packageIDs) > 0 {
		var packages []models.ProductPackage
		h.DB.Select("package_id, name").Where("package_id IN ?", packageIDs).Find(&packages)
		for _, p := range packages { packageNames[p.PackageID] = p.Name }
	}

	result := make([]PreviewItem, 0, len(items))
	for _, it := range items {
		qty := 1
		if it.Quantity.Valid { qty = int(it.Quantity.Int64) }
		up := 0.0
		if it.UnitPrice.Valid { up = it.UnitPrice.Float64 }
		lt := 0.0
		if it.LineTotal.Valid { lt = it.LineTotal.Float64 }
		pi := PreviewItem{ItemID: it.ItemID, RawText: it.RawProductText, Quantity: qty, UnitPrice: up, LineTotal: lt}
		if it.MappedProductID.Valid {
			pi.TargetType = "product"
			pi.TargetID = int(it.MappedProductID.Int64)
			pi.Name = productNames[pi.TargetID]
			if pi.Name == "" { pi.Name = fmt.Sprintf("Produkt #%d", pi.TargetID) }
		} else if it.MappedPackageID.Valid {
			pi.TargetType = "package"
			pi.TargetID = int(it.MappedPackageID.Int64)
			pi.Name = packageNames[pi.TargetID]
			if pi.Name == "" { pi.Name = fmt.Sprintf("Paket #%d", pi.TargetID) }
		}
		result = append(result, pi)
	}
	totalAmount := 0.0
	if extraction.TotalAmount.Valid { totalAmount = extraction.TotalAmount.Float64 }
	c.JSON(http.StatusOK, gin.H{
		"extraction_id": extraction.ExtractionID,
		"items":         result,
		"total_amount":  totalAmount,
		"item_count":    len(result),
	})
}

// CreateMappingAPI creates a standalone product mapping (no extraction context).
// POST /api/v1/pdf/mappings-create
// Body: { "pdf_text": "...", "product_id": 42 }
func (h *PDFHandler) CreateMappingAPI(c *gin.Context) {
	var req struct {
		PDFText   string `json:"pdf_text" binding:"required"`
		ProductID int    `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pdf_text and product_id are required"})
		return
	}
	var product models.Product
	if err := h.DB.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	userID := int64(0)
	if uid, exists := c.Get("userid"); exists {
		if id, ok := uid.(int64); ok { userID = id }
	}
	if err := h.Mapper.SaveMapping(req.PDFText, req.ProductID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save mapping"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "product_name": product.Name})
}

// UpdatePackageMappingAPI updates the target package for a package mapping.
// PUT /api/v1/pdf/package-mappings/:id
// Body: { "package_id": 7 }
func (h *PDFHandler) UpdatePackageMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}
	var req struct {
		PackageID int `json:"package_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package_id is required"})
		return
	}
	var pkg models.ProductPackage
	if err := h.DB.First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}
	result := h.DB.Model(&models.PDFPackageMapping{}).
		Where("mapping_id = ?", mappingID).
		Updates(map[string]interface{}{"package_id": req.PackageID, "mapping_type": "manual", "is_active": true})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "target_name": pkg.Name})
}

// DeletePackageMappingAPI soft-deletes a package mapping.
// DELETE /api/v1/pdf/package-mappings/:id
func (h *PDFHandler) DeletePackageMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}
	if h.PackageMapper == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Package mapper not available"})
		return
	}
	if err := h.PackageMapper.DeleteMapping(mappingID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateCustomerMappingAPI updates the target customer for a customer mapping.
// PUT /api/v1/pdf/customer-mappings/:id
// Body: { "customer_id": 3 }
func (h *PDFHandler) UpdateCustomerMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}
	var req struct {
		CustomerID int `json:"customer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CustomerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id is required"})
		return
	}
	var customer models.Customer
	if err := h.DB.First(&customer, req.CustomerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	result := h.DB.Model(&models.PDFCustomerMapping{}).
		Where("mapping_id = ?", mappingID).
		Updates(map[string]interface{}{"customer_id": req.CustomerID, "mapping_type": "manual", "is_active": true})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "target_name": customer.GetDisplayName()})
}

// DeleteCustomerMappingAPI soft-deletes a customer mapping.
// DELETE /api/v1/pdf/customer-mappings/:id
func (h *PDFHandler) DeleteCustomerMappingAPI(c *gin.Context) {
	mappingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mapping ID"})
		return
	}
	result := h.DB.Model(&models.PDFCustomerMapping{}).
		Where("mapping_id = ?", mappingID).
		Update("is_active", false)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateProductQuick creates a product and optionally N device stubs.
// POST /api/v1/pdf/product-quick-create
// Body: { "name": "...", "description": "...", "category_id": 1, "item_cost_per_day": 25.00, "device_count": 3 }
func (h *PDFHandler) CreateProductQuick(c *gin.Context) {
	var req struct {
		Name           string  `json:"name" binding:"required"`
		Description    string  `json:"description"`
		CategoryID     *int    `json:"category_id"`
		ItemCostPerDay float64 `json:"item_cost_per_day"`
		DeviceCount    int     `json:"device_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use raw SQL to avoid GORM column-name quoting issues with the products table
	type productRow struct {
		ProductID int `gorm:"column:productid"`
	}
	var row productRow
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	if err := h.DB.Raw(
		"INSERT INTO products (name, categoryid, description, itemcostperday) VALUES (?, ?, ?, ?) RETURNING productid",
		req.Name, req.CategoryID, desc, req.ItemCostPerDay,
	).Scan(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	devicesCreated := 0
	if req.DeviceCount > 0 && req.DeviceCount <= 500 {
		for i := 1; i <= req.DeviceCount; i++ {
			deviceID := fmt.Sprintf("P%d-%04d", row.ProductID, i)
			h.DB.Exec("INSERT INTO devices (deviceid, productid, status) VALUES (?, ?, 'free')", deviceID, row.ProductID)
			devicesCreated++
		}
	}
	c.JSON(http.StatusCreated, gin.H{
		"product_id":      row.ProductID,
		"name":            req.Name,
		"devices_created": devicesCreated,
	})
}

// CreatePackageQuick creates a product package.
// POST /api/v1/pdf/package-quick-create
// Body: { "name": "...", "description": "...", "code": "..." }
func (h *PDFHandler) CreatePackageQuick(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Code        string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	type pkgRow struct {
		ID int `gorm:"column:id"`
	}
	var row pkgRow
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	if err := h.DB.Raw(
		"INSERT INTO product_packages (name, description, package_code) VALUES (?, ?, ?) RETURNING id",
		req.Name, desc, req.Code,
	).Scan(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create package"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"package_id": row.ID,
		"name":       req.Name,
	})
}

// CreateRentalEquipmentQuick creates a rental equipment entry (Mietprodukt).
// POST /api/v1/pdf/rental-equipment-quick-create
// Body: { "name": "...", "supplier": "...", "rental_price": 0, "customer_price": 0, "category": "...", "description": "...", "notes": "..." }
func (h *PDFHandler) CreateRentalEquipmentQuick(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Supplier      string  `json:"supplier" binding:"required"`
		RentalPrice   float64 `json:"rental_price"`
		CustomerPrice float64 `json:"customer_price"`
		Category      string  `json:"category"`
		Description   string  `json:"description"`
		Notes         string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	type reRow struct {
		ID int `gorm:"column:id"`
	}
	var row reRow
	var desc, notes, category *string
	if req.Description != "" { desc = &req.Description }
	if req.Notes != "" { notes = &req.Notes }
	if req.Category != "" { category = &req.Category }
	if err := h.DB.Raw(
		`INSERT INTO rental_equipment (name, supplier, rental_price, customer_price, category, description, notes, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, true) RETURNING id`,
		req.Name, req.Supplier, req.RentalPrice, req.CustomerPrice, category, desc, notes,
	).Scan(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rental equipment"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"rental_equipment_id": row.ID,
		"name":                req.Name,
	})
}
