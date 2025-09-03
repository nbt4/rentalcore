package repository

import "go-barcode-webapp/internal/models"

type StatusRepository struct {
	db *Database
}

func NewStatusRepository(db *Database) *StatusRepository {
	return &StatusRepository{db: db}
}

func (r *StatusRepository) List() ([]models.Status, error) {
	var statuses []models.Status
	err := r.db.Find(&statuses).Error
	return statuses, err
}

func (r *StatusRepository) GetByID(id uint) (*models.Status, error) {
	var status models.Status
	err := r.db.First(&status, id).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}