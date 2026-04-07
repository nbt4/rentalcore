package handlers

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/services/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db             *gorm.DB
	uploadPath     string
	maxFileSize    int64
	allowedTypes   map[string]bool
	useNextcloud   bool
	ncClient       *storage.NextcloudClient
	ncBasePath     string
	backfillOnBoot bool
	assignedRoot   string
	unassignedRoot string
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	// Create upload directory if it doesn't exist
	uploadPath := "uploads"
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		panic("Failed to create upload directory: " + err.Error())
	}

	allowedTypes := map[string]bool{
		"application/pdf":                    true,
		"application/octet-stream":           true, // generic fallback browsers often send
		"image/jpeg":                         true,
		"image/jpg":                          true,
		"image/png":                          true,
		"image/gif":                          true,
		"image/webp":                         true,
		"text/plain":                         true,
		"application/msword":                 true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel":           true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	}

	assignedRoot := strings.Trim(trimQuotes(os.Getenv("FILEPOOL_ASSIGNED_ROOT")), "/")
	if assignedRoot == "" {
		assignedRoot = "assigned"
	}
	unassignedRoot := strings.Trim(trimQuotes(os.Getenv("FILEPOOL_UNASSIGNED_ROOT")), "/")
	if unassignedRoot == "" {
		unassignedRoot = "unassigned"
	}

	h := &DocumentHandler{
		db:             db,
		uploadPath:     uploadPath,
		maxFileSize:    10 * 1024 * 1024, // 10MB
		allowedTypes:   allowedTypes,
		useNextcloud:   false,
		ncBasePath:     trimQuotes(strings.Trim(os.Getenv("NEXTCLOUD_WEBDAV_BASE_PATH"), "/")),
		backfillOnBoot: strings.ToLower(os.Getenv("NEXTCLOUD_BACKFILL_ON_START")) != "false",
		assignedRoot:   assignedRoot,
		unassignedRoot: unassignedRoot,
	}

	ncURL := trimQuotes(strings.TrimSpace(os.Getenv("NEXTCLOUD_WEBDAV_URL")))
	ncUser := trimQuotes(strings.TrimSpace(os.Getenv("NEXTCLOUD_WEBDAV_USER")))
	ncPass := trimQuotes(strings.TrimSpace(os.Getenv("NEXTCLOUD_WEBDAV_PASSWORD")))

	if ncURL != "" && ncUser != "" && ncPass != "" {
		client, err := storage.NewNextcloudClient(ncURL, ncUser, ncPass, h.ncBasePath)
		if err != nil {
			panic("Failed to configure Nextcloud WebDAV client: " + err.Error())
		}
		h.useNextcloud = true
		h.ncClient = client
		if h.ncBasePath == "" {
			h.ncBasePath = "rentalcore-filepool"
		}
		if h.backfillOnBoot {
			go h.backfillLocalFiles()
			go h.backfillRemoteFiles()
		}
	}

	return h
}

// ================================================================
// DOCUMENT MANAGEMENT
// ================================================================

// ListDocuments displays documents for an entity or all documents
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	entityType := c.Query("entityType")
	entityID := c.Query("entityID")

	var documents []models.Document
	query := h.db.Preload("Uploader").Preload("Signatures").Order("uploaded_at DESC")

	// If entity parameters are provided, filter by them
	if entityType != "" && entityID != "" {
		query = query.Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	}

	result := query.Find(&documents)

	if result.Error != nil {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title": "Error",
			"error": "Failed to load documents",
			"user":  user,
		})
		return
	}

	user, _ := GetCurrentUser(c)
	title := "All Documents"
	if entityType != "" && entityID != "" {
		title = "Documents"
	}

	c.HTML(http.StatusOK, "documents_list.html", gin.H{
		"title":      title,
		"user":       user,
		"documents":  documents,
		"entityType": entityType,
		"entityID":   entityID,
	})
}

// UploadDocumentForm shows the document upload form
func (h *DocumentHandler) UploadDocumentForm(c *gin.Context) {
	entityType := c.Query("entityType")
	entityID := c.Query("entityID")

	if entityType == "" || entityID == "" {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Error",
			"error": "Entity type and ID are required",
			"user":  user,
		})
		return
	}

	user, _ := GetCurrentUser(c)
	c.HTML(http.StatusOK, "document_upload_form.html", gin.H{
		"title":      "Upload Document",
		"user":       user,
		"entityType": entityType,
		"entityID":   entityID,
	})
}

// UploadDocument handles file uploads
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	entityType := c.PostForm("entityType")
	entityID := c.PostForm("entityID")
	documentType := c.PostForm("documentType")
	description := c.PostForm("description")
	isPublic := c.PostForm("isPublic") == "true"

	if entityType == "" || entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity type and ID are required"})
		return
	}

	currentUser, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > h.maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File size exceeds maximum limit of %d MB", h.maxFileSize/(1024*1024)),
		})
		return
	}

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !h.allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
		return
	}

	// Generate unique filename
	filename := h.generateUniqueFilename(header.Filename)

	var finalPath string
	var checksum string
	var size int64

	if h.useNextcloud {
		buf := bytes.Buffer{}
		if _, err := io.Copy(&buf, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
		size = int64(buf.Len())
		if size > h.maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File size exceeds maximum limit of %d MB", h.maxFileSize/(1024*1024)),
			})
			return
		}
		finalPath = h.remotePath(entityType, entityID, filename)
		if err := h.ncClient.Upload(finalPath, bytes.NewReader(buf.Bytes()), contentType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to Nextcloud: " + err.Error()})
			return
		}
		checksum = h.calculateBufferChecksum(buf.Bytes())
	} else {
		// Create directory structure if needed
		entityDir := filepath.Join(h.uploadPath, entityType, entityID)
		if err := os.MkdirAll(entityDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
			return
		}

		finalPath = filepath.Join(entityDir, filename)

		// Save file
		if err := h.saveUploadedFile(file, finalPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Calculate checksum
		checksum, err = h.calculateFileChecksum(finalPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate checksum"})
			return
		}
		info, _ := os.Stat(finalPath)
		if info != nil {
			size = info.Size()
		}
	}

	// Save document record
	document := models.Document{
		EntityType:       entityType,
		EntityID:         entityID,
		Filename:         filename,
		OriginalFilename: header.Filename,
		FilePath:         finalPath,
		FileSize:         size,
		MimeType:         contentType,
		DocumentType:     documentType,
		Description:      description,
		UploadedBy:       &currentUser.UserID,
		UploadedAt:       time.Now(),
		IsPublic:         isPublic,
		Version:          1,
		Checksum:         checksum,
	}

	if err := h.db.Create(&document).Error; err != nil {
		// Clean up uploaded file on database error
		os.Remove(finalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document record"})
		return
	}

	accept := c.GetHeader("Accept")
	ct := c.ContentType()
	if strings.Contains(accept, "text/html") || strings.Contains(ct, "multipart/form-data") {
		// Return a simple popup then redirect back to the pool
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusCreated, `<script>alert('Upload erfolgreich: %s'); window.location.href='/documents/pool';</script>`, header.Filename)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Document uploaded successfully",
		"documentID": document.DocumentID,
		"filename":   filename,
	})
}

// DownloadDocument serves a document for download
func (h *DocumentHandler) DownloadDocument(c *gin.Context) {
	documentID := c.Param("id")

	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load document"})
		}
		return
	}

	// Check if file exists
	if h.useNextcloud && h.isRemote(document.FilePath) {
		reader, contentType, err := h.ncClient.Download(h.stripRemotePrefix(document.FilePath))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in storage"})
			return
		}
		defer reader.Close()

		if contentType != "" {
			c.Header("Content-Type", contentType)
		} else {
			c.Header("Content-Type", document.MimeType)
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", document.OriginalFilename))
		c.Stream(func(w io.Writer) bool {
			io.Copy(w, reader)
			return false
		})
		return
	}

	if _, err := os.Stat(document.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Set headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", document.OriginalFilename))
	c.Header("Content-Type", document.MimeType)

	c.File(document.FilePath)
}

// ViewDocument displays a document inline (for images, PDFs, etc.)
func (h *DocumentHandler) ViewDocument(c *gin.Context) {
	documentID := c.Param("id")

	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load document"})
		}
		return
	}

	// Check if file exists
	if h.useNextcloud && h.isRemote(document.FilePath) {
		reader, contentType, err := h.ncClient.Download(h.stripRemotePrefix(document.FilePath))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found in storage"})
			return
		}
		defer reader.Close()

		if contentType != "" {
			c.Header("Content-Type", contentType)
		} else {
			c.Header("Content-Type", document.MimeType)
		}
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", document.OriginalFilename))
		c.Stream(func(w io.Writer) bool {
			io.Copy(w, reader)
			return false
		})
		return
	}

	if _, err := os.Stat(document.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Set headers for inline display
	c.Header("Content-Type", document.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", document.OriginalFilename))

	c.File(document.FilePath)
}

// DeleteDocument removes a document
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	documentID := c.Param("id")

	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load document"})
		}
		return
	}

	// Delete file from disk
	if h.useNextcloud && h.isRemote(document.FilePath) {
		if err := h.ncClient.Delete(h.stripRemotePrefix(document.FilePath)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from storage"})
			return
		}
	} else {
		if err := os.Remove(document.FilePath); err != nil && !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from disk"})
			return
		}
	}

	// Delete database record
	if err := h.db.Delete(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// GetDocument retrieves document details
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	documentID := c.Param("id")

	var document models.Document
	result := h.db.Preload("Uploader").Preload("Signatures").First(&document, documentID)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load document"})
		}
		return
	}

	c.JSON(http.StatusOK, document)
}

// ================================================================
// DIGITAL SIGNATURES
// ================================================================

// SignatureForm shows the digital signature form
func (h *DocumentHandler) SignatureForm(c *gin.Context) {
	documentID := c.Param("id")

	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title": "Error",
			"error": "Document not found",
			"user":  user,
		})
		return
	}

	user, _ := GetCurrentUser(c)
	c.HTML(http.StatusOK, "signature_form.html", gin.H{
		"title":    "Sign Document",
		"user":     user,
		"document": document,
	})
}

// AddSignature adds a digital signature to a document
func (h *DocumentHandler) AddSignature(c *gin.Context) {
	documentID := c.Param("id")

	var request struct {
		SignerName    string `json:"signerName" binding:"required"`
		SignerEmail   string `json:"signerEmail"`
		SignerRole    string `json:"signerRole"`
		SignatureData string `json:"signatureData" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify document exists
	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Generate verification code
	verificationCode := h.generateVerificationCode()

	// Create signature record
	signature := models.DigitalSignature{
		DocumentID:       document.DocumentID,
		SignerName:       request.SignerName,
		SignerEmail:      request.SignerEmail,
		SignerRole:       request.SignerRole,
		SignatureData:    request.SignatureData,
		SignedAt:         time.Now(),
		IPAddress:        c.ClientIP(),
		VerificationCode: verificationCode,
		IsVerified:       true, // Auto-verify for now
	}

	if err := h.db.Create(&signature).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save signature"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":          "Document signed successfully",
		"signatureID":      signature.SignatureID,
		"verificationCode": verificationCode,
	})
}

// VerifySignature verifies a digital signature
func (h *DocumentHandler) VerifySignature(c *gin.Context) {
	signatureID := c.Param("id")
	verificationCode := c.Query("code")

	var signature models.DigitalSignature
	result := h.db.Preload("Document").First(&signature, signatureID)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	if signature.VerificationCode != verificationCode {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verified":    true,
		"signedAt":    signature.SignedAt,
		"signerName":  signature.SignerName,
		"signerEmail": signature.SignerEmail,
		"document":    signature.Document.OriginalFilename,
	})
}

// ================================================================
// UTILITY FUNCTIONS
// ================================================================

func (h *DocumentHandler) generateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("%d_%s%s", timestamp, randomHex, ext)
}

func (h *DocumentHandler) saveUploadedFile(file multipart.File, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return err
}

func (h *DocumentHandler) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (h *DocumentHandler) calculateBufferChecksum(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func (h *DocumentHandler) generateVerificationCode() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return strings.ToUpper(hex.EncodeToString(randomBytes))
}

func (h *DocumentHandler) respondError(c *gin.Context, status int, msg string) {
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "text/html") {
		user, _ := GetCurrentUser(c)
		c.HTML(status, "error.html", gin.H{
			"title": "Error",
			"error": msg,
			"user":  user,
		})
		return
	}
	c.JSON(status, gin.H{"error": msg})
}

// ================================================================
// API ENDPOINTS
// ================================================================

// ListDocumentsAPI returns documents as JSON
func (h *DocumentHandler) ListDocumentsAPI(c *gin.Context) {
	entityType := c.Query("entityType")
	entityID := c.Query("entityID")

	var documents []models.Document
	query := h.db.Preload("Uploader").Preload("Signatures")

	if entityType != "" && entityID != "" {
		query = query.Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	}

	if err := query.Order("uploaded_at DESC").Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"count":     len(documents),
	})
}

// GetDocumentStats returns document statistics
func (h *DocumentHandler) GetDocumentStats(c *gin.Context) {
	var stats struct {
		TotalDocuments  int64            `json:"totalDocuments"`
		TotalSize       int64            `json:"totalSize"`
		DocumentsByType map[string]int64 `json:"documentsByType"`
		SignedDocuments int64            `json:"signedDocuments"`
		RecentUploads   int64            `json:"recentUploads"`
	}

	// Total documents
	h.db.Model(&models.Document{}).Count(&stats.TotalDocuments)

	// Total size
	h.db.Model(&models.Document{}).Select("COALESCE(SUM(file_size), 0)").Scan(&stats.TotalSize)

	// Documents by type
	stats.DocumentsByType = make(map[string]int64)
	var typeStats []struct {
		DocumentType string `json:"document_type"`
		Count        int64  `json:"count"`
	}
	h.db.Model(&models.Document{}).
		Select("document_type, COUNT(*) as count").
		Group("document_type").
		Scan(&typeStats)

	for _, stat := range typeStats {
		stats.DocumentsByType[stat.DocumentType] = stat.Count
	}

	// Signed documents
	h.db.Model(&models.Document{}).
		Joins("INNER JOIN digital_signatures ON documents.documentID = digital_signatures.documentID").
		Count(&stats.SignedDocuments)

	// Recent uploads (last 7 days)
	weekAgo := time.Now().AddDate(0, 0, -7)
	h.db.Model(&models.Document{}).
		Where("uploaded_at > ?", weekAgo).
		Count(&stats.RecentUploads)

	c.JSON(http.StatusOK, stats)
}

// ListFilePool groups documents by assignment status.
func (h *DocumentHandler) ListFilePool(c *gin.Context) {
	// Auto-sync from Nextcloud before listing
	h.SyncFromNextcloud()

	var documents []models.Document
	if err := h.db.Preload("Uploader").Order("uploaded_at DESC").Find(&documents).Error; err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to load documents")
		return
	}

	assigned, unused := h.splitPoolDocuments(documents)
	assignedSize := h.sumFileSizes(assigned)
	unusedSize := h.sumFileSizes(unused)
	assignedCount := len(assigned)
	unusedCount := len(unused)

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "text/html") {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusOK, "documents_pool.html", gin.H{
			"title":         "File Pool",
			"user":          user,
			"assigned":      assigned,
			"unused":        unused,
			"assignedSize":  assignedSize,
			"unusedSize":    unusedSize,
			"assignedCount": assignedCount,
			"unusedCount":   unusedCount,
			"poolFolders": gin.H{
				"assigned":   h.assignedRoot,
				"unassigned": h.unassignedRoot,
				"base":       h.ncBasePath,
			},
			"nextcloudEnabled": h.useNextcloud,
			"currentPage":      "documents_pool",
			"timestamp":        time.Now().Unix(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assigned":      assigned,
		"unused":        unused,
		"assignedSize":  assignedSize,
		"unusedSize":    unusedSize,
		"assignedCount": assignedCount,
		"unusedCount":   unusedCount,
	})
}

// SyncFromNextcloud scans Nextcloud folders and creates database entries for files not yet tracked.
// This enables "live" detection of files added directly via Nextcloud or Windows Explorer.
func (h *DocumentHandler) SyncFromNextcloud() (int, error) {
	if !h.useNextcloud || h.ncClient == nil {
		return 0, nil
	}

	synced := 0

	// Sync unassigned folder
	unassignedFiles, err := h.ncClient.List(h.unassignedRoot)
	if err == nil {
		for _, entry := range unassignedFiles {
			if entry.IsDir || entry.Size == 0 {
				continue
			}

			filename := path.Base(entry.Path)
			remotePath := "nextcloud:" + strings.TrimLeft(entry.Path, "/")

			// Check if already in database
			var count int64
			h.db.Model(&models.Document{}).Where("file_path = ?", remotePath).Count(&count)
			if count > 0 {
				continue
			}

			// Determine document type from extension
			docType := h.guessDocumentType(filename)
			mimeType := h.guessMimeType(filename)

			// Create database entry
			doc := models.Document{
				EntityType:       "system",
				EntityID:         "unassigned",
				Filename:         filename,
				OriginalFilename: filename,
				FilePath:         remotePath,
				FileSize:         entry.Size,
				MimeType:         mimeType,
				DocumentType:     docType,
				Description:      "Auto-synced from Nextcloud",
				UploadedAt:       entry.ModTime,
				IsPublic:         false,
				Version:          1,
			}

			if err := h.db.Create(&doc).Error; err == nil {
				synced++
			}
		}
	}

	// Sync assigned folder (recursively)
	assignedFiles, err := h.ncClient.List(h.assignedRoot)
	if err == nil {
		for _, entry := range assignedFiles {
			if entry.IsDir || entry.Size == 0 {
				continue
			}

			filename := path.Base(entry.Path)
			remotePath := "nextcloud:" + strings.TrimLeft(entry.Path, "/")

			// Check if already in database
			var count int64
			h.db.Model(&models.Document{}).Where("file_path = ?", remotePath).Count(&count)
			if count > 0 {
				continue
			}

			// Parse entity type and ID from path: assigned/<entityType>/<entityID>/filename
			pathParts := strings.Split(strings.TrimPrefix(entry.Path, h.assignedRoot+"/"), "/")
			entityType := "system"
			entityID := "unknown"
			if len(pathParts) >= 3 {
				entityType = pathParts[0]
				entityID = pathParts[1]
			}

			docType := h.guessDocumentType(filename)
			mimeType := h.guessMimeType(filename)

			doc := models.Document{
				EntityType:       entityType,
				EntityID:         entityID,
				Filename:         filename,
				OriginalFilename: filename,
				FilePath:         remotePath,
				FileSize:         entry.Size,
				MimeType:         mimeType,
				DocumentType:     docType,
				Description:      "Auto-synced from Nextcloud",
				UploadedAt:       entry.ModTime,
				IsPublic:         false,
				Version:          1,
			}

			if err := h.db.Create(&doc).Error; err == nil {
				synced++
			}
		}
	}

	return synced, nil
}

// SyncFilePool is an HTTP endpoint to manually trigger Nextcloud sync
// GET /api/v1/documents/sync
func (h *DocumentHandler) SyncFilePool(c *gin.Context) {
	synced, err := h.SyncFromNextcloud()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"synced":  synced,
		"message": fmt.Sprintf("Synced %d new files from Nextcloud", synced),
	})
}

// Helper functions for sync
func (h *DocumentHandler) guessDocumentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "contract"
	case ".jpg", ".jpeg", ".png", ".gif":
		return "photo"
	case ".doc", ".docx":
		return "manual"
	default:
		return "other"
	}
}

func (h *DocumentHandler) guessMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

// Helpers
func (h *DocumentHandler) remotePath(entityType, entityID, filename string) string {
	// Store unassigned files in a flat "<unassignedRoot>/" directory; everything else under "<assignedRoot>/<entityType>/<entityID>"
	var rel string
	if entityType == "system" && strings.ToLower(entityID) == "unassigned" {
		rel = path.Join(h.unassignedRoot, filename)
	} else {
		rel = path.Join(h.assignedRoot, entityType, entityID, filename)
	}
	if h.useNextcloud {
		return "nextcloud:" + strings.TrimLeft(rel, "/")
	}
	return rel
}

func (h *DocumentHandler) stripRemotePrefix(p string) string {
	return strings.TrimPrefix(p, "nextcloud:")
}

func (h *DocumentHandler) isRemote(p string) bool {
	return strings.HasPrefix(p, "nextcloud:")
}

// Backfill existing local files into Nextcloud storage.
func (h *DocumentHandler) backfillLocalFiles() {
	if !h.useNextcloud {
		return
	}

	var documents []models.Document
	if err := h.db.Find(&documents).Error; err != nil {
		return
	}

	for _, doc := range documents {
		if h.isRemote(doc.FilePath) || doc.FilePath == "" {
			continue
		}
		file, err := os.Open(doc.FilePath)
		if err != nil {
			continue
		}
		info, _ := file.Stat()
		buf := bytes.Buffer{}
		io.Copy(&buf, file)
		file.Close()

		remote := h.remotePath(doc.EntityType, doc.EntityID, doc.Filename)
		if err := h.ncClient.Upload(remote, bytes.NewReader(buf.Bytes()), doc.MimeType); err != nil {
			continue
		}
		doc.FilePath = remote
		if info != nil {
			doc.FileSize = info.Size()
		}
		h.db.Model(&doc).Updates(map[string]interface{}{
			"file_path": doc.FilePath,
			"file_size": doc.FileSize,
		})
		_ = os.Remove(doc.FilePath)
	}
}

// backfillRemoteFiles imports files that exist in Nextcloud but not in the DB.
func (h *DocumentHandler) backfillRemoteFiles() {
	if !h.useNextcloud {
		return
	}

	entries, err := h.ncClient.List("")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir {
			continue
		}

		relPath := strings.TrimLeft(entry.Path, "/")
		entityType, entityID, ok := h.parseRemoteEntity(relPath)
		if !ok {
			continue
		}

		filePath := "nextcloud:" + relPath

		var count int64
		if err := h.db.Model(&models.Document{}).Where("file_path = ?", filePath).Count(&count).Error; err != nil {
			continue
		}
		if count > 0 {
			continue // already indexed
		}

		filename := path.Base(relPath)
		now := time.Now()
		uploadedAt := entry.ModTime
		if uploadedAt.IsZero() {
			uploadedAt = now
		}

		doc := models.Document{
			EntityType:       entityType,
			EntityID:         entityID,
			Filename:         filename,
			OriginalFilename: filename,
			FilePath:         filePath,
			FileSize:         entry.Size,
			MimeType:         chooseMime(entry.ContentType),
			DocumentType:     "other",
			Description:      "Imported from Nextcloud",
			UploadedAt:       uploadedAt,
			Version:          1,
			IsPublic:         false,
		}

		_ = h.db.Create(&doc).Error
	}
}

func chooseMime(m string) string {
	if strings.TrimSpace(m) == "" {
		return "application/octet-stream"
	}
	return m
}

// parseRemoteEntity maps a Nextcloud relative path to entityType/entityID using
// the configurable storage layout:
// - <unassignedRoot>/<file>
// - <assignedRoot>/<entityType>/<entityID>/<file>
func (h *DocumentHandler) parseRemoteEntity(relPath string) (string, string, bool) {
	segments := strings.Split(strings.TrimLeft(relPath, "/"), "/")
	if len(segments) < 2 {
		return "", "", false
	}
	if h.unassignedRoot != "" && segments[0] == h.unassignedRoot {
		return "system", "unassigned", true
	}
	if h.assignedRoot != "" && segments[0] == h.assignedRoot && len(segments) >= 4 {
		return segments[1], segments[2], true
	}
	return "", "", false
}

func (h *DocumentHandler) splitPoolDocuments(documents []models.Document) ([]models.Document, []models.Document) {
	assigned := []models.Document{}
	unused := []models.Document{}

	for _, doc := range documents {
		if h.isUnassignedPoolDoc(doc) {
			unused = append(unused, doc)
			continue
		}
		if h.isAssignedPoolDoc(doc) {
			assigned = append(assigned, doc)
		}
	}

	return assigned, unused
}

func (h *DocumentHandler) isUnassignedPoolDoc(doc models.Document) bool {
	if strings.ToLower(doc.EntityType) != "system" || strings.ToLower(doc.EntityID) != "unassigned" {
		return false
	}
	return h.folderMatches(doc.FilePath, h.unassignedRoot)
}

func (h *DocumentHandler) isAssignedPoolDoc(doc models.Document) bool {
	if strings.ToLower(doc.EntityType) == "system" && strings.ToLower(doc.EntityID) == "unassigned" {
		return false
	}
	return h.folderMatches(doc.FilePath, h.assignedRoot)
}

func (h *DocumentHandler) folderMatches(filePath, folder string) bool {
	folder = strings.Trim(folder, "/")
	if folder == "" {
		return false
	}
	if !h.useNextcloud {
		return true
	}
	normalized := h.normalizeFilePath(filePath)
	return normalized == folder || strings.HasPrefix(normalized, folder+"/")
}

func (h *DocumentHandler) normalizeFilePath(p string) string {
	p = strings.TrimPrefix(p, "nextcloud:")
	p = strings.TrimLeft(p, "/")
	return p
}

func (h *DocumentHandler) sumFileSizes(docs []models.Document) int64 {
	var total int64
	for _, doc := range docs {
		total += doc.FileSize
	}
	return total
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	return s
}

// ================================================================
// DOCUMENT ASSIGNMENT METHODS (File Pool as Single Source of Truth)
// ================================================================

// AssignDocumentToJob changes a document's entity_type/entity_id to assign it to a job
// This also moves the file from unassigned to assigned storage if using Nextcloud
func (h *DocumentHandler) AssignDocumentToJob(documentID uint, jobID uint) error {
	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		return fmt.Errorf("document not found: %v", err)
	}

	oldEntityType := document.EntityType
	oldEntityID := document.EntityID

	// Update the document assignment
	document.EntityType = "job"
	document.EntityID = fmt.Sprintf("%d", jobID)

	// If using Nextcloud, move the file to the assigned folder structure
	if h.useNextcloud && h.isRemote(document.FilePath) {
		newPath := h.remotePath("job", document.EntityID, document.Filename)
		oldPath := h.stripRemotePrefix(document.FilePath)

		// Only move if paths are different
		if oldPath != h.stripRemotePrefix(newPath) {
			if err := h.ncClient.Move(oldPath, h.stripRemotePrefix(newPath)); err != nil {
				return fmt.Errorf("failed to move file in storage: %v", err)
			}
			document.FilePath = newPath
		}
	} else if !h.useNextcloud {
		// For local storage, move the file to the job directory
		oldPath := document.FilePath
		newDir := filepath.Join(h.uploadPath, "job", document.EntityID)
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return fmt.Errorf("failed to create job directory: %v", err)
		}
		newPath := filepath.Join(newDir, document.Filename)

		if oldPath != newPath {
			if err := os.Rename(oldPath, newPath); err != nil {
				// Try copy instead
				if err := h.copyFile(oldPath, newPath); err != nil {
					return fmt.Errorf("failed to move file: %v", err)
				}
				os.Remove(oldPath)
			}
			document.FilePath = newPath
		}
	}

	// Save the updated document
	if err := h.db.Save(&document).Error; err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	// Log the assignment
	fmt.Printf("[INFO] Document %d assigned from %s/%s to job/%d\n", documentID, oldEntityType, oldEntityID, jobID)

	return nil
}

// UnassignDocument sets a document back to system/unassigned status
func (h *DocumentHandler) UnassignDocument(documentID uint) error {
	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		return fmt.Errorf("document not found: %v", err)
	}

	oldEntityType := document.EntityType
	oldEntityID := document.EntityID

	// Update to unassigned
	document.EntityType = "system"
	document.EntityID = "unassigned"

	// If using Nextcloud, move the file to the unassigned folder
	if h.useNextcloud && h.isRemote(document.FilePath) {
		newPath := h.remotePath("system", "unassigned", document.Filename)
		oldPath := h.stripRemotePrefix(document.FilePath)

		if oldPath != h.stripRemotePrefix(newPath) {
			if err := h.ncClient.Move(oldPath, h.stripRemotePrefix(newPath)); err != nil {
				return fmt.Errorf("failed to move file in storage: %v", err)
			}
			document.FilePath = newPath
		}
	} else if !h.useNextcloud {
		// For local storage, move to unassigned directory
		oldPath := document.FilePath
		newDir := filepath.Join(h.uploadPath, "system", "unassigned")
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return fmt.Errorf("failed to create unassigned directory: %v", err)
		}
		newPath := filepath.Join(newDir, document.Filename)

		if oldPath != newPath {
			if err := os.Rename(oldPath, newPath); err != nil {
				if err := h.copyFile(oldPath, newPath); err != nil {
					return fmt.Errorf("failed to move file: %v", err)
				}
				os.Remove(oldPath)
			}
			document.FilePath = newPath
		}
	}

	if err := h.db.Save(&document).Error; err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	fmt.Printf("[INFO] Document %d unassigned from %s/%s\n", documentID, oldEntityType, oldEntityID)

	return nil
}

// GetDocumentByID retrieves a document by its ID
func (h *DocumentHandler) GetDocumentByID(documentID uint) (*models.Document, error) {
	var document models.Document
	if err := h.db.First(&document, documentID).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

// GetJobDocuments returns all documents assigned to a specific job
func (h *DocumentHandler) GetJobDocuments(jobID uint) ([]models.Document, error) {
	var documents []models.Document
	if err := h.db.Where("entity_type = ? AND entity_id = ?", "job", fmt.Sprintf("%d", jobID)).
		Order("uploaded_at DESC").
		Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

// CreateDocumentFromFile creates a new document in the File Pool from file data
// This is used when uploading PDFs for OCR processing
func (h *DocumentHandler) CreateDocumentFromFile(
	filename string,
	originalFilename string,
	filePath string,
	fileSize int64,
	mimeType string,
	uploadedBy *uint,
	entityType string,
	entityID string,
) (*models.Document, error) {
	// Calculate checksum if local file
	var checksum string
	if !h.isRemote(filePath) {
		var err error
		checksum, err = h.calculateFileChecksum(filePath)
		if err != nil {
			checksum = ""
		}
	}

	document := models.Document{
		EntityType:       entityType,
		EntityID:         entityID,
		Filename:         filename,
		OriginalFilename: originalFilename,
		FilePath:         filePath,
		FileSize:         fileSize,
		MimeType:         mimeType,
		DocumentType:     "other",
		Description:      "Uploaded via PDF OCR",
		UploadedBy:       uploadedBy,
		UploadedAt:       time.Now(),
		IsPublic:         false,
		Version:          1,
		Checksum:         checksum,
	}

	if err := h.db.Create(&document).Error; err != nil {
		return nil, fmt.Errorf("failed to create document: %v", err)
	}

	return &document, nil
}

// copyFile copies a file from src to dst
func (h *DocumentHandler) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// GetDB returns the database handle for use by other handlers
func (h *DocumentHandler) GetDB() *gorm.DB {
	return h.db
}

// UseNextcloud returns whether Nextcloud storage is enabled
func (h *DocumentHandler) UseNextcloud() bool {
	return h.useNextcloud
}

// GetNextcloudClient returns the Nextcloud client
func (h *DocumentHandler) GetNextcloudClient() *storage.NextcloudClient {
	return h.ncClient
}

// GetUploadPath returns the local upload path
func (h *DocumentHandler) GetUploadPath() string {
	return h.uploadPath
}

// GetRemotePath generates the remote path for a document
func (h *DocumentHandler) GetRemotePath(entityType, entityID, filename string) string {
	return h.remotePath(entityType, entityID, filename)
}
