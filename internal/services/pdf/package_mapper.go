package pdf

import (
	"database/sql"
	"errors"
	"time"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

// PackageMapper handles package mapping between PDF text and database packages
type PackageMapper struct {
	DB *gorm.DB
}

// NewPackageMapper creates a new package mapper instance
func NewPackageMapper(db *gorm.DB) *PackageMapper {
	return &PackageMapper{
		DB: db,
	}
}

// SaveMapping saves a manual package mapping
func (m *PackageMapper) SaveMapping(pdfText string, packageID int, userID int64) error {
	normalized := normalizeProductText(pdfText)

	confidence := 100.0
	lastUsed := time.Now()
	normalizedVal := nullStringPtr(sql.NullString{String: normalized, Valid: normalized != ""})
	createdBy := nullIntPtr(sql.NullInt64{Int64: userID, Valid: userID > 0})

	query := `
		INSERT INTO pdf_package_mappings
			(pdf_package_text, normalized_text, package_id, mapping_type, confidence_score, usage_count, last_used_at, created_by, is_active)
		VALUES
			($1, $2, $3, 'manual', $4, 1, $5, $6, true)
		ON CONFLICT (pdf_package_text) DO UPDATE SET
			normalized_text = EXCLUDED.normalized_text,
			package_id = EXCLUDED.package_id,
			mapping_type = 'manual',
			confidence_score = EXCLUDED.confidence_score,
			usage_count = pdf_package_mappings.usage_count + 1,
			last_used_at = EXCLUDED.last_used_at,
			is_active = true
	`

	return m.DB.Exec(query,
		pdfText,
		normalizedVal,
		packageID,
		confidence,
		lastUsed,
		createdBy,
	).Error
}

// GetAllMappings retrieves all active package mappings
func (m *PackageMapper) GetAllMappings() ([]models.PDFPackageMapping, error) {
	var mappings []models.PDFPackageMapping
	err := m.DB.Where("is_active = ?", true).
		Order("usage_count DESC, updated_at DESC").
		Find(&mappings).Error
	return mappings, err
}

// DeleteMapping deletes or deactivates a package mapping
func (m *PackageMapper) DeleteMapping(mappingID uint64) error {
	result := m.DB.Model(&models.PDFPackageMapping{}).
		Where("mapping_id = ?", mappingID).
		Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// LookupSavedMapping finds an existing saved mapping for the given text
func (m *PackageMapper) LookupSavedMapping(packageText string) (*models.ProductPackage, error) {
	if m == nil {
		return nil, nil
	}
	var existingMapping models.PDFPackageMapping
	err := m.DB.Where("pdf_package_text = ? AND is_active = ?", packageText, true).
		First(&existingMapping).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		normalized := normalizeProductText(packageText)
		if normalized == "" {
			return nil, nil
		}
		if err := m.DB.Where("normalized_text = ? AND is_active = ?", normalized, true).
			First(&existingMapping).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, nil
		}
	}

	// Mark usage
	m.DB.Model(&existingMapping).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": time.Now(),
	})

	var pkg models.ProductPackage
	if err := m.DB.First(&pkg, existingMapping.PackageID).Error; err != nil {
		return nil, err
	}

	return &pkg, nil
}
