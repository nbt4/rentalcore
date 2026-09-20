package repository

import (
	"errors"
	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrRequirementJobNotFound     = errors.New("requirement job not found")
	ErrRequirementProductNotFound = errors.New("requirement product not found")
	ErrRequirementAlreadyExists   = errors.New("job product requirement already exists")
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

// CreateRequirement adds one new product requirement without modifying any
// existing requirement. This is the additive API used by guided integrations.
func (r *RequirementRepository) CreateRequirement(req *models.JobProductRequirement) error {
	return r.db.DB.Transaction(func(tx *gorm.DB) error {
		var jobCount int64
		if err := tx.Model(&models.Job{}).Where("jobid = ? AND deleted_at IS NULL", req.JobID).Count(&jobCount).Error; err != nil {
			return err
		}
		if jobCount == 0 {
			return ErrRequirementJobNotFound
		}

		var productCount int64
		if err := tx.Model(&models.Product{}).Where("productid = ? AND lifecycle_status = ?", req.ProductID, "active").Count(&productCount).Error; err != nil {
			return err
		}
		if productCount == 0 {
			return ErrRequirementProductNotFound
		}

		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "job_id"}, {Name: "product_id"}},
			DoNothing: true,
		}).Create(req)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRequirementAlreadyExists
		}
		return nil
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
