package services

import (
	"database/sql"
	"fmt"
	"time"

	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

// JobHistoryService handles logging of job changes
type JobHistoryService struct {
	db *gorm.DB
}

// NewJobHistoryService creates a new job history service
func NewJobHistoryService(db *gorm.DB) *JobHistoryService {
	return &JobHistoryService{db: db}
}

// LogJobCreation logs the creation of a new job
func (s *JobHistoryService) LogJobCreation(jobID uint, userID *uint, ipAddress, userAgent string) error {
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "created",
		ChangedAt:  time.Now(),
		Description: sql.NullString{
			String: "Job created",
			Valid:  true,
		},
	}

	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}

	return s.db.Create(&history).Error
}

// LogJobUpdate logs updates to a job by comparing old and new values
func (s *JobHistoryService) LogJobUpdate(oldJob, newJob *models.Job, userID *uint, ipAddress, userAgent string) error {
	changes := s.detectChanges(oldJob, newJob)
	if len(changes) == 0 {
		return nil // No changes detected
	}

	histories := make([]models.JobHistory, 0, len(changes))
	now := time.Now()

	for _, change := range changes {
		history := models.JobHistory{
			JobID:      newJob.JobID,
			ChangeType: "updated",
			ChangedAt:  now,
			FieldName:  sql.NullString{String: change.Field, Valid: true},
			OldValue:   sql.NullString{String: change.OldValue, Valid: change.OldValue != ""},
			NewValue:   sql.NullString{String: change.NewValue, Valid: change.NewValue != ""},
			Description: sql.NullString{
				String: fmt.Sprintf("%s changed from '%s' to '%s'",
					models.FormatFieldName(change.Field),
					change.OldValue,
					change.NewValue),
				Valid: true,
			},
		}

		if userID != nil {
			history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
		}
		if ipAddress != "" {
			history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
		}
		if userAgent != "" {
			history.UserAgent = sql.NullString{String: userAgent, Valid: true}
		}

		histories = append(histories, history)
	}

	return s.db.Create(&histories).Error
}

// LogStatusChange logs a status change specifically
func (s *JobHistoryService) LogStatusChange(jobID uint, oldStatusID, newStatusID uint, userID *uint, ipAddress, userAgent string) error {
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "status_changed",
		ChangedAt:  time.Now(),
		FieldName:  sql.NullString{String: "statusID", Valid: true},
		OldValue:   sql.NullString{String: fmt.Sprintf("%d", oldStatusID), Valid: true},
		NewValue:   sql.NullString{String: fmt.Sprintf("%d", newStatusID), Valid: true},
		Description: sql.NullString{
			String: fmt.Sprintf("Status changed"),
			Valid:  true,
		},
	}

	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}

	return s.db.Create(&history).Error
}

// LogDeviceAdded logs when a device is added to a job
func (s *JobHistoryService) LogDeviceAdded(jobID uint, deviceID uint, userID *uint, ipAddress, userAgent string) error {
	return s.logDeviceChange(jobID, deviceID, "device_added", "Device added to job", userID, ipAddress, userAgent)
}

// LogDeviceRemoved logs when a device is removed from a job
func (s *JobHistoryService) LogDeviceRemoved(jobID uint, deviceID uint, userID *uint, ipAddress, userAgent string) error {
	return s.logDeviceChange(jobID, deviceID, "device_removed", "Device removed from job", userID, ipAddress, userAgent)
}

// LogRequirementAdded records additive material planning changes made through
// the UI or a delegated suite integration.
func (s *JobHistoryService) LogRequirementAdded(jobID, productID uint, quantity int, productName string, userID *uint, ipAddress, userAgent string) error {
	description := fmt.Sprintf("Product requirement added: %d × %s", quantity, productName)
	if productName == "" {
		description = fmt.Sprintf("Product requirement added: %d × product %d", quantity, productID)
	}
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "requirement_added",
		ChangedAt:  time.Now(),
		FieldName:  sql.NullString{String: "product_requirements", Valid: true},
		NewValue:   sql.NullString{String: fmt.Sprintf("product_id=%d,quantity=%d", productID, quantity), Valid: true},
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
	}
	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}
	return s.db.Create(&history).Error
}

// LogRequirementUpdated records both quantities so the change can be reviewed
// and manually reversed without reconstructing earlier job state.
func (s *JobHistoryService) LogRequirementUpdated(jobID, productID uint, oldQuantity, newQuantity int, productName string, userID *uint, ipAddress, userAgent string) error {
	description := fmt.Sprintf("Product requirement updated: %d → %d × %s", oldQuantity, newQuantity, productName)
	if productName == "" {
		description = fmt.Sprintf("Product requirement updated: product %d, %d → %d", productID, oldQuantity, newQuantity)
	}
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "requirement_updated",
		ChangedAt:  time.Now(),
		FieldName:  sql.NullString{String: "product_requirements", Valid: true},
		OldValue:   sql.NullString{String: fmt.Sprintf("product_id=%d,quantity=%d", productID, oldQuantity), Valid: true},
		NewValue:   sql.NullString{String: fmt.Sprintf("product_id=%d,quantity=%d", productID, newQuantity), Valid: true},
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
	}
	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}
	return s.db.Create(&history).Error
}

func (s *JobHistoryService) logDeviceChange(jobID, deviceID uint, changeType, description string, userID *uint, ipAddress, userAgent string) error {
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: changeType,
		ChangedAt:  time.Now(),
		NewValue:   sql.NullString{String: fmt.Sprintf("%d", deviceID), Valid: true},
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
	}

	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}

	return s.db.Create(&history).Error
}

// LogFileAdded logs when a file is added to a job
func (s *JobHistoryService) LogFileAdded(jobID uint, filename string, userID *uint, ipAddress, userAgent string) error {
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "file_added",
		ChangedAt:  time.Now(),
		FieldName:  sql.NullString{String: "attachment", Valid: true},
		NewValue:   sql.NullString{String: filename, Valid: true},
		Description: sql.NullString{
			String: fmt.Sprintf("File added: %s", filename),
			Valid:  true,
		},
	}

	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}

	return s.db.Create(&history).Error
}

// LogFileRemoved logs when a file is removed from a job
func (s *JobHistoryService) LogFileRemoved(jobID uint, filename string, userID *uint, ipAddress, userAgent string) error {
	history := models.JobHistory{
		JobID:      jobID,
		ChangeType: "file_removed",
		ChangedAt:  time.Now(),
		FieldName:  sql.NullString{String: "attachment", Valid: true},
		OldValue:   sql.NullString{String: filename, Valid: true},
		Description: sql.NullString{
			String: fmt.Sprintf("File removed: %s", filename),
			Valid:  true,
		},
	}

	if userID != nil {
		history.UserID = sql.NullInt64{Int64: int64(*userID), Valid: true}
	}
	if ipAddress != "" {
		history.IPAddress = sql.NullString{String: ipAddress, Valid: true}
	}
	if userAgent != "" {
		history.UserAgent = sql.NullString{String: userAgent, Valid: true}
	}

	return s.db.Create(&history).Error
}

// GetJobHistory retrieves the history for a specific job
func (s *JobHistoryService) GetJobHistory(jobID uint) ([]models.JobHistoryEntry, error) {
	var histories []models.JobHistory
	err := s.db.Where("job_id = ?", jobID).
		Preload("User").
		Order("changed_at DESC").
		Find(&histories).Error

	if err != nil {
		return nil, err
	}

	entries := make([]models.JobHistoryEntry, len(histories))
	for i, h := range histories {
		entry := models.JobHistoryEntry{
			HistoryID:  h.HistoryID,
			JobID:      h.JobID,
			ChangedAt:  h.ChangedAt,
			ChangeType: h.ChangeType,
		}

		if h.UserID.Valid {
			userID := uint(h.UserID.Int64)
			entry.UserID = &userID
			if h.User != nil {
				entry.UserName = h.User.GetDisplayName()
			}
		} else {
			entry.UserName = "System"
		}

		if h.FieldName.Valid {
			entry.FieldName = &h.FieldName.String
		}
		if h.OldValue.Valid {
			entry.OldValue = &h.OldValue.String
		}
		if h.NewValue.Valid {
			entry.NewValue = &h.NewValue.String
		}
		if h.Description.Valid {
			entry.Description = h.Description.String
		}
		if h.IPAddress.Valid {
			entry.IPAddress = &h.IPAddress.String
		}

		entries[i] = entry
	}

	return entries, nil
}

// Change represents a field change
type Change struct {
	Field    string
	OldValue string
	NewValue string
}

// detectChanges compares two job objects and returns a list of changes
func (s *JobHistoryService) detectChanges(oldJob, newJob *models.Job) []Change {
	changes := []Change{}

	// Compare basic fields
	if oldJob.CustomerID != newJob.CustomerID {
		changes = append(changes, Change{
			Field:    "customerID",
			OldValue: fmt.Sprintf("%d", oldJob.CustomerID),
			NewValue: fmt.Sprintf("%d", newJob.CustomerID),
		})
	}

	if oldJob.StatusID != newJob.StatusID {
		changes = append(changes, Change{
			Field:    "statusID",
			OldValue: fmt.Sprintf("%d", oldJob.StatusID),
			NewValue: fmt.Sprintf("%d", newJob.StatusID),
		})
	}

	// Compare nullable fields
	changes = append(changes, s.compareNullableUint(oldJob.JobCategoryID, newJob.JobCategoryID, "jobcategoryID")...)
	changes = append(changes, s.compareNullableString(oldJob.Description, newJob.Description, "description")...)

	// Compare discount
	if oldJob.Discount != newJob.Discount {
		changes = append(changes, Change{
			Field:    "discount",
			OldValue: fmt.Sprintf("%.2f", oldJob.Discount),
			NewValue: fmt.Sprintf("%.2f", newJob.Discount),
		})
	}

	if oldJob.DiscountType != newJob.DiscountType {
		changes = append(changes, Change{
			Field:    "discount_type",
			OldValue: oldJob.DiscountType,
			NewValue: newJob.DiscountType,
		})
	}

	// Compare dates
	changes = append(changes, s.compareNullableTime(oldJob.StartDate, newJob.StartDate, "startDate")...)
	changes = append(changes, s.compareNullableTime(oldJob.EndDate, newJob.EndDate, "endDate")...)

	return changes
}

func (s *JobHistoryService) compareNullableUint(old, new *uint, field string) []Change {
	oldStr := "null"
	newStr := "null"
	if old != nil {
		oldStr = fmt.Sprintf("%d", *old)
	}
	if new != nil {
		newStr = fmt.Sprintf("%d", *new)
	}
	if oldStr != newStr {
		return []Change{{Field: field, OldValue: oldStr, NewValue: newStr}}
	}
	return nil
}

func (s *JobHistoryService) compareNullableString(old, new *string, field string) []Change {
	oldStr := "null"
	newStr := "null"
	if old != nil {
		oldStr = *old
	}
	if new != nil {
		newStr = *new
	}
	if oldStr != newStr {
		return []Change{{Field: field, OldValue: oldStr, NewValue: newStr}}
	}
	return nil
}

func (s *JobHistoryService) compareNullableTime(old, new *time.Time, field string) []Change {
	oldStr := "null"
	newStr := "null"
	if old != nil {
		oldStr = old.Format("2006-01-02")
	}
	if new != nil {
		newStr = new.Format("2006-01-02")
	}
	if oldStr != newStr {
		return []Change{{Field: field, OldValue: oldStr, NewValue: newStr}}
	}
	return nil
}
