package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type EmployeeRepository struct{ db *gorm.DB }

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) List() ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Preload("Skills").Order("last_name, first_name").Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) ListActive() ([]models.Employee, error) {
	var employees []models.Employee
	err := r.db.Preload("Skills").Where("is_active = true").
		Order("last_name, first_name").Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetByID(id uint) (*models.Employee, error) {
	var e models.Employee
	err := r.db.Preload("Skills").First(&e, id).Error
	return &e, err
}

func (r *EmployeeRepository) Create(e *models.Employee, skillIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(e).Error; err != nil {
			return err
		}
		return r.replaceSkills(tx, e, skillIDs)
	})
}

func (r *EmployeeRepository) Update(e *models.Employee, skillIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(e).Error; err != nil {
			return err
		}
		return r.replaceSkills(tx, e, skillIDs)
	})
}

func (r *EmployeeRepository) Delete(id uint) error {
	return r.db.Delete(&models.Employee{}, id).Error
}

func (r *EmployeeRepository) replaceSkills(tx *gorm.DB, e *models.Employee, skillIDs []uint) error {
	if err := tx.Exec("DELETE FROM employee_skills WHERE employee_id = ?", e.ID).Error; err != nil {
		return err
	}
	if len(skillIDs) == 0 {
		return nil
	}
	var skills []models.Skill
	if err := tx.Where("id IN ?", skillIDs).Find(&skills).Error; err != nil {
		return err
	}
	return tx.Model(e).Association("Skills").Replace(skills)
}
