package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type SkillRepository struct{ db *gorm.DB }

func NewSkillRepository(db *gorm.DB) *SkillRepository {
	return &SkillRepository{db: db}
}

func (r *SkillRepository) List() ([]models.Skill, error) {
	var skills []models.Skill
	err := r.db.Order("category, name").Find(&skills).Error
	return skills, err
}

func (r *SkillRepository) GetByID(id uint) (*models.Skill, error) {
	var s models.Skill
	err := r.db.First(&s, id).Error
	return &s, err
}

func (r *SkillRepository) Create(s *models.Skill) error {
	return r.db.Create(s).Error
}

func (r *SkillRepository) Update(s *models.Skill) error {
	return r.db.Save(s).Error
}

func (r *SkillRepository) Delete(id uint) error {
	return r.db.Delete(&models.Skill{}, id).Error
}
