package repository

import (
	"go-barcode-webapp/internal/models"
)

type JobCategoryRepository struct {
	db *Database
}

func NewJobCategoryRepository(db *Database) *JobCategoryRepository {
	return &JobCategoryRepository{db: db}
}

func (r *JobCategoryRepository) Create(category *models.JobCategory) error {
	return r.db.Create(category).Error
}

func (r *JobCategoryRepository) GetByID(id uint) (*models.JobCategory, error) {
	var category models.JobCategory
	err := r.db.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *JobCategoryRepository) List() ([]models.JobCategory, error) {
	var categories []models.JobCategory
	err := r.db.Find(&categories).Error
	return categories, err
}

func (r *JobCategoryRepository) Update(category *models.JobCategory) error {
	return r.db.Save(category).Error
}

func (r *JobCategoryRepository) Delete(id uint) error {
	return r.db.Delete(&models.JobCategory{}, id).Error
}