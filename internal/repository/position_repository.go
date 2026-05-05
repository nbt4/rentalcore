package repository

import (
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type PositionRepository struct {
	db *Database
}

func NewPositionRepository(db *Database) *PositionRepository {
	return &PositionRepository{db: db}
}

func (r *PositionRepository) GetByJobID(jobID uint) ([]models.JobPosition, error) {
	var positions []models.JobPosition
	err := r.db.Where("job_id = ?", jobID).
		Preload("Product").
		Preload("ServiceItem").
		Preload("Devices").
		Order("sort_order ASC, position_id ASC").
		Find(&positions).Error
	return positions, err
}

func (r *PositionRepository) GetByID(positionID uint) (*models.JobPosition, error) {
	var pos models.JobPosition
	err := r.db.Where("position_id = ?", positionID).
		Preload("Product").
		Preload("ServiceItem").
		Preload("Devices").
		First(&pos).Error
	if err != nil {
		return nil, err
	}
	return &pos, nil
}

func (r *PositionRepository) Create(pos *models.JobPosition) error {
	return r.db.Create(pos).Error
}

func (r *PositionRepository) Update(pos *models.JobPosition) error {
	return r.db.Save(pos).Error
}

func (r *PositionRepository) Delete(positionID uint) error {
	return r.db.Where("position_id = ?", positionID).Delete(&models.JobPosition{}).Error
}

func (r *PositionRepository) Reorder(jobID uint, positionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range positionIDs {
			if err := tx.Model(&models.JobPosition{}).
				Where("position_id = ? AND job_id = ?", id, jobID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PositionRepository) AssignDevice(positionID uint, deviceID string, scannedBy string) error {
	dev := models.JobPositionDevice{
		PositionID: positionID,
		DeviceID:   deviceID,
		ScannedBy:  scannedBy,
	}
	return r.db.Where("position_id = ? AND device_id = ?", positionID, deviceID).
		FirstOrCreate(&dev).Error
}

func (r *PositionRepository) RemoveDevice(positionID uint, deviceID string) error {
	return r.db.Where("position_id = ? AND device_id = ?", positionID, deviceID).
		Delete(&models.JobPositionDevice{}).Error
}

func (r *PositionRepository) GetPicklist(jobID uint) ([]models.JobPosition, error) {
	var positions []models.JobPosition
	err := r.db.Where("job_id = ? AND position_type = ?", jobID, "product").
		Preload("Product").
		Preload("Devices").
		Order("sort_order ASC, position_id ASC").
		Find(&positions).Error
	return positions, err
}

func (r *PositionRepository) GetNextSortOrder(jobID uint) (int, error) {
	var maxOrder *int
	err := r.db.Model(&models.JobPosition{}).
		Where("job_id = ?", jobID).
		Select("MAX(sort_order)").
		Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder + 1, nil
}
