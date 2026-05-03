package pdf

import (
	"database/sql"
	"errors"
	"time"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type RentalEquipmentResult struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type RentalMapper struct {
	DB *gorm.DB
}

func NewRentalMapper(db *gorm.DB) *RentalMapper {
	return &RentalMapper{DB: db}
}

func (m *RentalMapper) LookupSavedMapping(text string) (*RentalEquipmentResult, error) {
	if m == nil {
		return nil, nil
	}
	var mapping models.PDFRentalMapping
	err := m.DB.Where("pdf_rental_text = ? AND is_active = ?", text, true).
		First(&mapping).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		normalized := normalizeProductText(text)
		if normalized == "" {
			return nil, nil
		}
		if err := m.DB.Where("normalized_text = ? AND is_active = ?", normalized, true).
			First(&mapping).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, nil
		}
	}

	m.DB.Model(&mapping).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": time.Now(),
	})

	var name string
	err = m.DB.Raw("SELECT name FROM rental_equipment WHERE id = ? AND is_active = true", mapping.RentalEquipmentID).Scan(&name).Error
	if err != nil || name == "" {
		return nil, err
	}

	return &RentalEquipmentResult{ID: int64(mapping.RentalEquipmentID), Name: name}, nil
}

func (m *RentalMapper) FindBestMatch(text string) (*RentalEquipmentResult, float64, error) {
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
	if err := m.DB.Raw("SELECT id, name FROM rental_equipment WHERE is_active = true").Scan(&rows).Error; err != nil {
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
		return &RentalEquipmentResult{ID: best.ID, Name: best.Name}, bestScore, nil
	}

	return nil, 0, nil
}

func (m *RentalMapper) SaveMapping(pdfText string, rentalEquipmentID int, userID int64) error {
	normalized := normalizeProductText(pdfText)
	lastUsed := time.Now()
	normalizedVal := nullStringPtr(sql.NullString{String: normalized, Valid: normalized != ""})
	createdBy := nullIntPtr(sql.NullInt64{Int64: userID, Valid: userID > 0})

	query := `
		INSERT INTO pdf_rental_mappings
			(pdf_rental_text, normalized_text, rental_equipment_id, mapping_type, confidence_score, usage_count, last_used_at, created_by, is_active)
		VALUES
			($1, $2, $3, 'manual', 100, 1, $4, $5, true)
		ON CONFLICT (pdf_rental_text) DO UPDATE SET
			normalized_text = EXCLUDED.normalized_text,
			rental_equipment_id = EXCLUDED.rental_equipment_id,
			mapping_type = 'manual',
			confidence_score = 100,
			usage_count = pdf_rental_mappings.usage_count + 1,
			last_used_at = EXCLUDED.last_used_at,
			is_active = true
	`

	return m.DB.Exec(query,
		pdfText,
		normalizedVal,
		rentalEquipmentID,
		lastUsed,
		createdBy,
	).Error
}
