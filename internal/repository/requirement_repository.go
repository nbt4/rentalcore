package repository

import (
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type RequirementRepository struct {
	db *Database
}

func NewRequirementRepository(db *Database) *RequirementRepository {
	return &RequirementRepository{db: db}
}

// SaveRequirements replaces all requirements for the given job atomically.
// Pass an empty slice to clear all requirements.
func (r *RequirementRepository) SaveRequirements(jobID uint, reqs []models.JobProductRequirement) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id = ?", jobID).Delete(&models.JobProductRequirement{}).Error; err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		return tx.Create(&reqs).Error
	})
}

// GetByJobID returns all requirements for a job, with product preloaded.
func (r *RequirementRepository) GetByJobID(jobID uint) ([]models.JobProductRequirement, error) {
	var reqs []models.JobProductRequirement
	err := r.db.Where("job_id = ?", jobID).
		Preload("Product").
		Order("id ASC").
		Find(&reqs).Error
	return reqs, err
}
