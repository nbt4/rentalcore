package repository

import (
	"database/sql"

	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"

	"go-barcode-webapp/internal/logger"
)

type CableRepository struct {
	db *Database
}

func NewCableRepository(db *Database) *CableRepository {
	return &CableRepository{db: db}
}

func (r *CableRepository) Create(cable *models.Cable) error {
	return r.db.Create(cable).Error
}

func (r *CableRepository) GetByID(id int) (*models.Cable, error) {
	var cable models.Cable
	err := r.db.Preload("Connector1Info").Preload("Connector2Info").Preload("TypeInfo").First(&cable, id).Error
	if err != nil {
		return nil, err
	}
	return &cable, nil
}

func (r *CableRepository) Update(cable *models.Cable) error {
	return r.db.Save(cable).Error
}

func (r *CableRepository) Delete(id int) error {
	return r.db.Delete(&models.Cable{}, id).Error
}

func (r *CableRepository) List(params *models.FilterParams) ([]models.Cable, error) {
	var cables []models.Cable

	query := r.db.Model(&models.Cable{}).
		Preload("Connector1Info").
		Preload("Connector2Info").
		Preload("TypeInfo")

	query = applyCableFilters(query, params)

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	query = query.Order("name ASC")

	err := query.Find(&cables).Error
	return cables, err
}

// ListGrouped returns cables grouped by specifications with count
func (r *CableRepository) ListGrouped(params *models.FilterParams) ([]models.CableGroup, error) {
	var groups []models.CableGroup

	// Build the base query for grouping
	query := r.db.Model(&models.Cable{}).
		Select("typ as type, connector1, connector2, length, mm2, name, COUNT(*) as count").
		Group("typ, connector1, connector2, length, mm2, name").
		Order("name ASC")

	query = applyCableFilters(query, params)

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	// Execute the grouping query
	err := query.Find(&groups).Error
	if err != nil {
		return nil, err
	}

	// Load relationship data for each group
	for i := range groups {
		// Load connector info
		if groups[i].Connector1 > 0 {
			var connector1 models.CableConnector
			if err := r.db.First(&connector1, groups[i].Connector1).Error; err == nil {
				groups[i].Connector1Info = &connector1
			}
		}

		if groups[i].Connector2 > 0 {
			var connector2 models.CableConnector
			if err := r.db.First(&connector2, groups[i].Connector2).Error; err == nil {
				groups[i].Connector2Info = &connector2
			}
		}

		// Load type info
		if groups[i].Type > 0 {
			var cableType models.CableType
			if err := r.db.First(&cableType, groups[i].Type).Error; err == nil {
				groups[i].TypeInfo = &cableType
			}
		}

		// Get sample cable IDs for this group
		var cableIDs []int
		whereClause := "typ = ? AND connector1 = ? AND connector2 = ? AND length = ? AND COALESCE(name, '') = COALESCE(?, '')"
		args := []interface{}{groups[i].Type, groups[i].Connector1, groups[i].Connector2, groups[i].Length, groups[i].Name}

		if groups[i].MM2 != nil {
			whereClause += " AND mm2 = ?"
			args = append(args, *groups[i].MM2)
		} else {
			whereClause += " AND mm2 IS NULL"
		}

		r.db.Model(&models.Cable{}).
			Select("cableID").
			Where(whereClause, args...).
			Pluck("cableID", &cableIDs)
		groups[i].CableIDs = cableIDs
	}

	return groups, nil
}

func (r *CableRepository) GetTotalCount() (int, error) {
	var count int64
	err := r.db.Model(&models.Cable{}).Count(&count).Error
	return int(count), err
}

func (r *CableRepository) GetGroupedTotalCount(params *models.FilterParams) (int, error) {
	query := applyCableFilters(r.db.Model(&models.Cable{}), params)

	var count int64
	err := query.Distinct("typ", "connector1", "connector2", "length", "mm2", "name").Count(&count).Error
	return int(count), err
}

// GetLengthBounds returns minimum and maximum cable lengths for slider defaults
func (r *CableRepository) GetLengthBounds() (float64, float64, error) {
	var result struct {
		MinLength sql.NullFloat64
		MaxLength sql.NullFloat64
	}

	if err := r.db.Model(&models.Cable{}).
		Select("MIN(length) AS min_length, MAX(length) AS max_length").
		Scan(&result).Error; err != nil {
		return 0, 0, err
	}

	min := 0.0
	max := 0.0
	if result.MinLength.Valid {
		min = result.MinLength.Float64
	}
	if result.MaxLength.Valid {
		max = result.MaxLength.Float64
	}

	return min, max, nil
}

// Get all cable types for forms
func (r *CableRepository) GetAllCableTypes() ([]models.CableType, error) {
	var types []models.CableType
	err := r.db.Order("name ASC").Find(&types).Error
	if err != nil {
		logger.LogInfo("❌ GetAllCableTypes error: %v", err)
		return nil, err
	}
	return types, nil
}

// Get all cable connectors for forms
func (r *CableRepository) GetAllCableConnectors() ([]models.CableConnector, error) {
	var connectors []models.CableConnector
	err := r.db.Order("name ASC").Find(&connectors).Error
	if err != nil {
		logger.LogInfo("❌ GetAllCableConnectors error: %v", err)
		return nil, err
	}
	return connectors, nil
}

func (r *CableRepository) GetConnectorPairings() (map[int][]int, error) {
	type pair struct {
		Connector1 int
		Connector2 int
	}

	var pairs []pair
	if err := r.db.Model(&models.Cable{}).
		Select("DISTINCT connector1, connector2").
		Find(&pairs).Error; err != nil {
		return nil, err
	}

	result := make(map[int][]int)
	for _, p := range pairs {
		result[p.Connector1] = appendUnique(result[p.Connector1], p.Connector2)
		result[p.Connector2] = appendUnique(result[p.Connector2], p.Connector1)
	}
	return result, nil
}

func appendUnique(slice []int, value int) []int {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

func applyCableFilters(query *gorm.DB, params *models.FilterParams) *gorm.DB {
	if params == nil {
		return query
	}

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("name LIKE ?", searchPattern)
	}

	if params.Connector1ID != nil && params.Connector2ID != nil {
		c1 := int(*params.Connector1ID)
		c2 := int(*params.Connector2ID)
		query = query.Where(
			"(connector1 = ? AND connector2 = ?) OR (connector1 = ? AND connector2 = ?)",
			c1,
			c2,
			c2,
			c1,
		)
	} else if params.Connector1ID != nil {
		c1 := int(*params.Connector1ID)
		query = query.Where("(connector1 = ?) OR (connector2 = ?)", c1, c1)
	} else if params.Connector2ID != nil {
		c2 := int(*params.Connector2ID)
		query = query.Where("(connector1 = ?) OR (connector2 = ?)", c2, c2)
	}

	if params.CableTypeID != nil {
		query = query.Where("typ = ?", int(*params.CableTypeID))
	}
	if params.MinLength != nil {
		query = query.Where("length >= ?", *params.MinLength)
	}
	if params.MaxLength != nil {
		query = query.Where("length <= ?", *params.MaxLength)
	}
	return query
}
