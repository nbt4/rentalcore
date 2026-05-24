package repository

import (
	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"
)

type JobEmployeeRepository struct{ db *gorm.DB }

func NewJobEmployeeRepository(db *gorm.DB) *JobEmployeeRepository {
	return &JobEmployeeRepository{db: db}
}

func (r *JobEmployeeRepository) ListForJob(jobID uint) ([]models.JobEmployee, error) {
	var je []models.JobEmployee
	err := r.db.Preload("Employee.Skills").Where("job_id = ?", jobID).Find(&je).Error
	return je, err
}

func (r *JobEmployeeRepository) Assign(jobID, employeeID uint, role *string) error {
	je := models.JobEmployee{JobID: jobID, EmployeeID: employeeID, Role: role}
	return r.db.Where(models.JobEmployee{JobID: jobID, EmployeeID: employeeID}).
		FirstOrCreate(&je).Error
}

func (r *JobEmployeeRepository) Remove(jobID, employeeID uint) error {
	return r.db.Where("job_id = ? AND employee_id = ?", jobID, employeeID).
		Delete(&models.JobEmployee{}).Error
}

func (r *JobEmployeeRepository) GetOne(jobID, employeeID uint) (*models.JobEmployee, error) {
	var je models.JobEmployee
	err := r.db.Preload("Employee").
		Where("job_id = ? AND employee_id = ?", jobID, employeeID).
		First(&je).Error
	return &je, err
}

func (r *JobEmployeeRepository) SaveM365EventID(jobID, employeeID uint, eventID string) error {
	return r.db.Model(&models.JobEmployee{}).
		Where("job_id = ? AND employee_id = ?", jobID, employeeID).
		Update("m365_event_id", eventID).Error
}

func (r *JobEmployeeRepository) ClearM365EventID(jobID, employeeID uint) error {
	return r.db.Model(&models.JobEmployee{}).
		Where("job_id = ? AND employee_id = ?", jobID, employeeID).
		Update("m365_event_id", nil).Error
}
