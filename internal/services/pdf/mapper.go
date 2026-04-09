package pdf

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

// ProductMapper handles product mapping between PDF text and database products
type ProductMapper struct {
	DB         *gorm.DB
	aliasCache *PackageAliasCache
}

// NewProductMapper creates a new product mapper instance
func NewProductMapper(db *gorm.DB, aliasCache *PackageAliasCache) *ProductMapper {
	return &ProductMapper{
		DB:         db,
		aliasCache: aliasCache,
	}
}

// FindBestMatch finds the best matching suggestion (product or package)
func (m *ProductMapper) FindBestMatch(productText string) (*models.ProductMappingSuggestion, error) {
	text := strings.TrimSpace(productText)
	if text == "" {
		return nil, nil
	}

	if suggestion, err := m.lookupSavedMapping(text); err != nil {
		return nil, err
	} else if suggestion != nil {
		return suggestion, nil
	}

	if m.aliasCache != nil {
		if suggestion := m.aliasCache.FindBestMatch(text); suggestion != nil {
			return suggestion, nil
		}
	}

	normalized := normalizeProductText(text)
	if normalized == "" {
		return nil, nil
	}

	var products []models.Product
	if err := m.DB.Find(&products).Error; err != nil {
		return nil, err
	}

	var bestMatch *models.Product
	bestScore := 0.0
	for i := range products {
		score := calculateSimilarity(normalized, normalizeProductText(products[i].Name))
		if score > bestScore {
			bestScore = score
			bestMatch = &products[i]
		}
	}

	if bestMatch != nil && bestScore >= 70.0 {
		return &models.ProductMappingSuggestion{
			RawProductText:   text,
			SuggestedProduct: bestMatch,
			Confidence:       bestScore,
			MappingType:      "fuzzy",
		}, nil
	}

	return nil, nil
}

func (m *ProductMapper) lookupSavedMapping(productText string) (*models.ProductMappingSuggestion, error) {
	if m == nil {
		return nil, nil
	}
	var existingMapping models.PDFProductMapping
	err := m.DB.Where("pdf_product_text = ? AND is_active = ?", productText, true).
		First(&existingMapping).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		normalized := normalizeProductText(productText)
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

	mapping, product, _, err := m.markMappingUsage(&existingMapping)
	if err != nil || mapping == nil || product == nil {
		return nil, err
	}

	return &models.ProductMappingSuggestion{
		RawProductText:   productText,
		SuggestedProduct: product,
		Confidence:       100,
		MappingType:      mapping.MappingType,
	}, nil
}

// SaveMapping saves a manual product mapping
func (m *ProductMapper) SaveMapping(pdfText string, productID int, userID int64) error {
	normalized := normalizeProductText(pdfText)

	confidence := 100.0
	lastUsed := time.Now()
	normalizedVal := nullStringPtr(sql.NullString{String: normalized, Valid: normalized != ""})
	createdBy := nullIntPtr(sql.NullInt64{Int64: userID, Valid: userID > 0})

	query := `
		INSERT INTO pdf_product_mappings
			(pdf_product_text, normalized_text, product_id, mapping_type, confidence_score, usage_count, last_used_at, created_by, is_active)
		VALUES
			($1, $2, $3, 'manual', $4, 1, $5, $6, true)
		ON CONFLICT (pdf_product_text) DO UPDATE SET
			normalized_text = EXCLUDED.normalized_text,
			product_id = EXCLUDED.product_id,
			mapping_type = 'manual',
			confidence_score = EXCLUDED.confidence_score,
			usage_count = pdf_product_mappings.usage_count + 1,
			last_used_at = EXCLUDED.last_used_at,
			is_active = true
	`

	return m.DB.Exec(query,
		pdfText,
		normalizedVal,
		productID,
		confidence,
		lastUsed,
		createdBy,
	).Error
}

// GetAllMappings retrieves all active mappings
func (m *ProductMapper) GetAllMappings() ([]models.PDFProductMapping, error) {
	var mappings []models.PDFProductMapping
	err := m.DB.Where("is_active = ?", true).
		Order("usage_count DESC, updated_at DESC").
		Find(&mappings).Error
	return mappings, err
}

// DeleteMapping deletes or deactivates a mapping
func (m *ProductMapper) DeleteMapping(mappingID uint64) error {
	result := m.DB.Model(&models.PDFProductMapping{}).
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

func (m *ProductMapper) markMappingUsage(mapping *models.PDFProductMapping) (*models.PDFProductMapping, *models.Product, float64, error) {
	m.DB.Model(mapping).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": time.Now(),
	})

	var product models.Product
	if err := m.DB.First(&product, mapping.ProductID).Error; err != nil {
		return nil, nil, 0, err
	}

	return mapping, &product, 100.0, nil
}

func nullStringPtr(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}

func nullIntPtr(value sql.NullInt64) interface{} {
	if value.Valid {
		return value.Int64
	}
	return nil
}

// normalizeProductText normalizes text for comparison
func normalizeProductText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace non-alphanumeric characters with spaces to keep word boundaries
	text = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, text)

	// Normalize whitespace
	text = strings.Join(strings.Fields(text), " ")

	return strings.TrimSpace(text)
}

// calculateSimilarity calculates similarity between two strings (0-100)
// Uses a simple approach: Levenshtein distance ratio
func calculateSimilarity(s1, s2 string) float64 {
	// Quick checks
	if s1 == s2 {
		return 100.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Check if one contains the other
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		shorter := len(s1)
		if len(s2) < shorter {
			shorter = len(s2)
		}
		longer := len(s1)
		if len(s2) > longer {
			longer = len(s2)
		}
		return float64(shorter) / float64(longer) * 100.0
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	// Convert to similarity percentage
	similarity := (1.0 - float64(distance)/float64(maxLen)) * 100.0
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// FindSimilarProducts finds products/packages similar to the given text
func (m *ProductMapper) FindSimilarProducts(productText string, limit int) ([]models.ProductMappingSuggestion, error) {
	if limit <= 0 {
		limit = 5
	}

	suggestions := make([]models.ProductMappingSuggestion, 0, limit)
	if m.aliasCache != nil {
		if matches := m.aliasCache.FindMatches(productText, limit); len(matches) > 0 {
			suggestions = append(suggestions, matches...)
		}
	}

	if len(suggestions) >= limit {
		return suggestions[:limit], nil
	}

	normalized := normalizeProductText(productText)
	if normalized == "" {
		return suggestions, nil
	}

	var products []models.Product
	if err := m.DB.Limit(500).Find(&products).Error; err != nil {
		return nil, err
	}

	type scored struct {
		score      float64
		suggestion models.ProductMappingSuggestion
	}
	candidates := make([]scored, 0, len(products))

	for i := range products {
		score := calculateSimilarity(normalized, normalizeProductText(products[i].Name))
		if score < 50.0 {
			continue
		}
		candidates = append(candidates, scored{
			score: score,
			suggestion: models.ProductMappingSuggestion{
				RawProductText:   productText,
				SuggestedProduct: &products[i],
				Confidence:       score,
				MappingType:      "fuzzy",
			},
		})
	}

	if len(candidates) == 0 {
		return suggestions, nil
	}

	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	for _, candidate := range candidates {
		if len(suggestions) >= limit {
			break
		}
		suggestions = append(suggestions, candidate.suggestion)
	}

	return suggestions, nil
}
