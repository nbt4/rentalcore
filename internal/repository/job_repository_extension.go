package repository

import (
	"fmt"
	"go-barcode-webapp/internal/models"
)

// FreeDevicesFromCompletedJobs removes device assignments from jobs with "cancelled" status
// and sets the device status back to "free". Paid jobs should retain their device assignments for records.
func (r *JobRepository) FreeDevicesFromCompletedJobs() error {
	// Find all jobs with "cancelled" status (NOT "paid" - paid jobs should keep device assignments)
	var cancelledJobs []models.Job
	err := r.db.Joins("JOIN status ON jobs.statusID = status.statusID").
		Where("status.status = ?", "cancelled").
		Find(&cancelledJobs).Error
	if err != nil {
		return fmt.Errorf("failed to find cancelled jobs: %v", err)
	}

	if len(cancelledJobs) == 0 {
		return nil // No cancelled jobs found
	}

	// Extract job IDs
	var jobIDs []uint
	for _, job := range cancelledJobs {
		jobIDs = append(jobIDs, job.JobID)
	}

	// Find all devices assigned to these jobs
	var jobDevices []models.JobDevice
	err = r.db.Where("jobID IN ?", jobIDs).Find(&jobDevices).Error
	if err != nil {
		return fmt.Errorf("failed to find devices in cancelled jobs: %v", err)
	}

	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Remove job device assignments
	if len(jobDevices) > 0 {
		err = tx.Where("jobID IN ?", jobIDs).Delete(&models.JobDevice{}).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to remove job device assignments: %v", err)
		}

		// Set device status back to "free"
		var deviceIDs []string
		for _, jd := range jobDevices {
			deviceIDs = append(deviceIDs, jd.DeviceID)
		}

		err = tx.Model(&models.Device{}).
			Where("deviceID IN ?", deviceIDs).
			Update("status", "free").Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update device status: %v", err)
		}
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}