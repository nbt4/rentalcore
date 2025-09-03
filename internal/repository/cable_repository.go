package repository

import (
	"log"
	"go-barcode-webapp/internal/models"
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

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("name LIKE ?", searchPattern)
	}

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

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("name LIKE ?", searchPattern)
	}

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

// Get all cable types for forms
func (r *CableRepository) GetAllCableTypes() ([]models.CableType, error) {
	var types []models.CableType
	err := r.db.Order("name ASC").Find(&types).Error
	if err != nil {
		log.Printf("❌ GetAllCableTypes error: %v", err)
		return nil, err
	}
	return types, nil
}

// Get all cable connectors for forms
func (r *CableRepository) GetAllCableConnectors() ([]models.CableConnector, error) {
	var connectors []models.CableConnector
	err := r.db.Order("name ASC").Find(&connectors).Error
	if err != nil {
		log.Printf("❌ GetAllCableConnectors error: %v", err)
		return nil, err
	}
	return connectors, nil
}