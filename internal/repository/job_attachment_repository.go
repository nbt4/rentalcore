package repository

import (
	"go-barcode-webapp/internal/models"
	"time"
)

type JobAttachmentRepository struct {
	db *Database
}

func NewJobAttachmentRepository(db *Database) *JobAttachmentRepository {
	return &JobAttachmentRepository{db: db}
}

// Create creates a new job attachment
func (r *JobAttachmentRepository) Create(attachment *models.JobAttachment) error {
	return r.db.Create(attachment).Error
}

// GetByID retrieves a job attachment by ID
func (r *JobAttachmentRepository) GetByID(attachmentID uint) (*models.JobAttachment, error) {
	var attachment models.JobAttachment
	err := r.db.Where("attachment_id = ? AND is_active = ?", attachmentID, true).
		Preload("Job").
		Preload("Uploader").
		First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// GetByJobID retrieves all attachments for a specific job
func (r *JobAttachmentRepository) GetByJobID(jobID uint) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	err := r.db.Where("job_id = ? AND is_active = ?", jobID, true).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at DESC").
		Find(&attachments).Error
	return attachments, err
}

// Update updates a job attachment
func (r *JobAttachmentRepository) Update(attachment *models.JobAttachment) error {
	return r.db.Save(attachment).Error
}

// Delete soft deletes a job attachment (sets is_active to false)
func (r *JobAttachmentRepository) Delete(attachmentID uint) error {
	return r.db.Model(&models.JobAttachment{}).
		Where("attachment_id = ?", attachmentID).
		Update("is_active", false).Error
}

// HardDelete permanently deletes a job attachment
func (r *JobAttachmentRepository) HardDelete(attachmentID uint) error {
	return r.db.Where("attachment_id = ?", attachmentID).Delete(&models.JobAttachment{}).Error
}

// GetByFilename retrieves a job attachment by filename
func (r *JobAttachmentRepository) GetByFilename(filename string) (*models.JobAttachment, error) {
	var attachment models.JobAttachment
	err := r.db.Where("filename = ? AND is_active = ?", filename, true).
		Preload("Job").
		Preload("Uploader").
		First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// GetTotalSizeByJobID returns the total file size for all attachments of a job
func (r *JobAttachmentRepository) GetTotalSizeByJobID(jobID uint) (int64, error) {
	var totalSize int64
	err := r.db.Model(&models.JobAttachment{}).
		Where("job_id = ? AND is_active = ?", jobID, true).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error
	return totalSize, err
}

// GetCountByJobID returns the number of attachments for a job
func (r *JobAttachmentRepository) GetCountByJobID(jobID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.JobAttachment{}).
		Where("job_id = ? AND is_active = ?", jobID, true).
		Count(&count).Error
	return count, err
}

// GetRecentUploads returns recently uploaded attachments
func (r *JobAttachmentRepository) GetRecentUploads(limit int) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	err := r.db.Where("is_active = ?", true).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at DESC").
		Limit(limit).
		Find(&attachments).Error
	return attachments, err
}

// GetByMimeType returns attachments filtered by MIME type
func (r *JobAttachmentRepository) GetByMimeType(jobID uint, mimeType string) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	err := r.db.Where("job_id = ? AND mime_type LIKE ? AND is_active = ?", jobID, mimeType+"%", true).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at DESC").
		Find(&attachments).Error
	return attachments, err
}

// GetUploadedByUser returns attachments uploaded by a specific user
func (r *JobAttachmentRepository) GetUploadedByUser(userID uint, limit int) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	err := r.db.Where("uploaded_by = ? AND is_active = ?", userID, true).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at DESC").
		Limit(limit).
		Find(&attachments).Error
	return attachments, err
}

// GetAttachmentsOlderThan returns attachments uploaded before a specific date
func (r *JobAttachmentRepository) GetAttachmentsOlderThan(date time.Time) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	err := r.db.Where("uploaded_at < ? AND is_active = ?", date, true).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at ASC").
		Find(&attachments).Error
	return attachments, err
}

// SearchAttachments searches attachments by filename or description
func (r *JobAttachmentRepository) SearchAttachments(jobID uint, searchTerm string) ([]models.JobAttachment, error) {
	var attachments []models.JobAttachment
	searchPattern := "%" + searchTerm + "%"

	err := r.db.Where("job_id = ? AND is_active = ? AND (original_filename LIKE ? OR description LIKE ?)",
		jobID, true, searchPattern, searchPattern).
		Preload("Job").
		Preload("Uploader").
		Order("uploaded_at DESC").
		Find(&attachments).Error
	return attachments, err
}