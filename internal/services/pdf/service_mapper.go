package pdf

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ServiceItemResult represents a matched service item
type ServiceItemResult struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ServiceMapper handles mapping between PDF text and service items
type ServiceMapper struct {
	DB *gorm.DB
}

// NewServiceMapper creates a new service mapper instance
func NewServiceMapper(db *gorm.DB) *ServiceMapper {
	return &ServiceMapper{DB: db}
}

// LookupSavedMapping looks up an exact or normalized saved mapping in pdf_service_mappings
func (m *ServiceMapper) LookupSavedMapping(text string) (*ServiceItemResult, error) {
	if m == nil {
		return nil, nil
	}

	type mapping struct {
		MappingID     uint64 `gorm:"column:mapping_id;primaryKey"`
		ServiceItemID int    `gorm:"column:service_item_id"`
		UsageCount    int    `gorm:"column:usage_count"`
	}

	var mp mapping
	err := m.DB.Table("pdf_service_mappings").
		Where("pdf_service_text = ? AND is_active = ?", text, true).
		First(&mp).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		normalized := normalizeProductText(text)
		if normalized == "" {
			return nil, nil
		}
		if err := m.DB.Table("pdf_service_mappings").
			Where("normalized_text = ? AND is_active = ?", normalized, true).
			First(&mp).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, nil
		}
	}

	// Bump usage count
	m.DB.Table("pdf_service_mappings").Where("mapping_id = ?", mp.MappingID).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": time.Now(),
	})

	var name string
	err = m.DB.Raw("SELECT name FROM service_items WHERE id = ? AND is_active = true", mp.ServiceItemID).Scan(&name).Error
	if err != nil || name == "" {
		return nil, err
	}

	return &ServiceItemResult{ID: int64(mp.ServiceItemID), Name: name}, nil
}

// FindBestMatch finds the best matching service item — first checks saved mappings, then fuzzy match
func (m *ServiceMapper) FindBestMatch(text string) (*ServiceItemResult, float64, error) {
	if m == nil {
		return nil, 0, nil
	}

	if result, err := m.LookupSavedMapping(text); err != nil {
		return nil, 0, err
	} else if result != nil {
		return result, 100.0, nil
	}

	normalized := normalizeProductText(text)
	if normalized == "" {
		return nil, 0, nil
	}

	type row struct {
		ID   int64
		Name string
	}
	var rows []row
	if err := m.DB.Raw("SELECT id, name FROM service_items WHERE is_active = true").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	var best *row
	bestScore := 0.0
	for i := range rows {
		score := calculateSimilarity(normalized, normalizeProductText(rows[i].Name))
		if score > bestScore {
			bestScore = score
			best = &rows[i]
		}
	}

	if best != nil && bestScore >= 75.0 {
		return &ServiceItemResult{ID: best.ID, Name: best.Name}, bestScore, nil
	}

	return nil, 0, nil
}

// SaveMapping persists a mapping between PDF text and a service item
func (m *ServiceMapper) SaveMapping(pdfText string, serviceItemID int, userID int64) error {
	normalized := normalizeProductText(pdfText)
	lastUsed := time.Now()
	normalizedVal := nullStringPtr(sql.NullString{String: normalized, Valid: normalized != ""})
	createdBy := nullIntPtr(sql.NullInt64{Int64: userID, Valid: userID > 0})

	query := `
		INSERT INTO pdf_service_mappings
			(pdf_service_text, normalized_text, service_item_id, mapping_type, confidence_score, usage_count, last_used_at, created_by, is_active)
		VALUES
			($1, $2, $3, 'manual', 100, 1, $4, $5, true)
		ON CONFLICT (pdf_service_text) DO UPDATE SET
			normalized_text = EXCLUDED.normalized_text,
			service_item_id = EXCLUDED.service_item_id,
			mapping_type = 'manual',
			confidence_score = 100,
			usage_count = pdf_service_mappings.usage_count + 1,
			last_used_at = EXCLUDED.last_used_at,
			is_active = true
	`

	return m.DB.Exec(query,
		pdfText,
		normalizedVal,
		serviceItemID,
		lastUsed,
		createdBy,
	).Error
}
