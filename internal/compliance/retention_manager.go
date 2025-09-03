package compliance

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// RetentionManager handles data retention policies according to German law
type RetentionManager struct {
	db *gorm.DB
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager(db *gorm.DB) (*RetentionManager, error) {
	return &RetentionManager{
		db: db,
	}, nil
}

// GetRetentionDate calculates the retention date for a document type
func (rm *RetentionManager) GetRetentionDate(documentType string) (time.Time, error) {
	var policy RetentionPolicy
	if err := rm.db.Where("document_type = ? AND is_active = ?", documentType, true).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Default to 10 years if no specific policy found
			return time.Now().AddDate(10, 0, 0), nil
		}
		return time.Time{}, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return time.Now().AddDate(policy.RetentionYears, 0, 0), nil
}

// CanAutoDelete checks if a document type can be automatically deleted
func (rm *RetentionManager) CanAutoDelete(documentType string) (bool, error) {
	var policy RetentionPolicy
	if err := rm.db.Where("document_type = ? AND is_active = ?", documentType, true).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Default to false for safety
			return false, nil
		}
		return false, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return policy.AutoDeleteAfter, nil
}

// CreateRetentionPolicy creates a new retention policy
func (rm *RetentionManager) CreateRetentionPolicy(policy *RetentionPolicy) error {
	// Check if policy already exists
	var existing RetentionPolicy
	if err := rm.db.Where("document_type = ?", policy.DocumentType).First(&existing).Error; err == nil {
		return fmt.Errorf("retention policy for document type %s already exists", policy.DocumentType)
	}

	return rm.db.Create(policy).Error
}

// UpdateRetentionPolicy updates an existing retention policy
func (rm *RetentionManager) UpdateRetentionPolicy(documentType string, updates *RetentionPolicy) error {
	return rm.db.Model(&RetentionPolicy{}).
		Where("document_type = ?", documentType).
		Updates(updates).Error
}

// GetRetentionPolicies gets all retention policies
func (rm *RetentionManager) GetRetentionPolicies() ([]RetentionPolicy, error) {
	var policies []RetentionPolicy
	if err := rm.db.Where("is_active = ?", true).Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to get retention policies: %w", err)
	}
	return policies, nil
}

// GetExpiringDocuments gets documents expiring within the specified duration
func (rm *RetentionManager) GetExpiringDocuments(within time.Duration) ([]GoBDRecord, error) {
	cutoff := time.Now().Add(within)
	
	var records []GoBDRecord
	if err := rm.db.Where("retention_date <= ?", cutoff).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get expiring documents: %w", err)
	}
	return records, nil
}

// PerformRetentionCleanup performs automated retention cleanup
func (rm *RetentionManager) PerformRetentionCleanup() (*RetentionCleanupReport, error) {
	report := &RetentionCleanupReport{
		StartTime: time.Now(),
	}

	// Get all expired records
	now := time.Now()
	var expiredRecords []GoBDRecord
	if err := rm.db.Where("retention_date < ?", now).Find(&expiredRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to find expired records: %w", err)
	}

	report.TotalExpired = len(expiredRecords)

	// Process each expired record
	for _, record := range expiredRecords {
		// Check if auto-deletion is allowed
		canDelete, err := rm.CanAutoDelete(record.DocumentType)
		if err != nil {
			log.Printf("Warning: Failed to check auto-delete policy for %s: %v", record.DocumentType, err)
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to check policy for %s: %v", record.DocumentType, err))
			continue
		}

		if !canDelete {
			report.SkippedAutoDelete++
			log.Printf("Auto-deletion disabled for document type %s, skipping record %d", record.DocumentType, record.ID)
			continue
		}

		// Mark for deletion (in practice, you might want to move to a "pending deletion" state first)
		if err := rm.markForDeletion(record); err != nil {
			log.Printf("Warning: Failed to mark record %d for deletion: %v", record.ID, err)
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to mark record %d for deletion: %v", record.ID, err))
			continue
		}

		report.MarkedForDeletion++
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	return report, nil
}

// markForDeletion marks a record for deletion (implementing safe deletion pattern)
func (rm *RetentionManager) markForDeletion(record GoBDRecord) error {
	// In production, you might want to:
	// 1. Move record to a "pending_deletion" table
	// 2. Create a deletion request for approval
	// 3. Wait for a grace period before actual deletion
	// 4. Log the deletion request in audit logs

	// For now, we'll just log the action
	log.Printf("Record %d (%s:%s) marked for deletion - retention period expired on %s", 
		record.ID, record.DocumentType, record.DocumentID, record.RetentionDate.Format("2006-01-02"))

	return nil
}

// GetRetentionStatus provides an overview of retention status
func (rm *RetentionManager) GetRetentionStatus() (*RetentionStatus, error) {
	status := &RetentionStatus{
		Policies: make(map[string]RetentionPolicyStatus),
	}

	// Get all policies
	policies, err := rm.GetRetentionPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to get retention policies: %w", err)
	}

	for _, policy := range policies {
		// Count documents for this policy
		var total int64
		if err := rm.db.Model(&GoBDRecord{}).
			Where("document_type = ?", policy.DocumentType).
			Count(&total).Error; err != nil {
			log.Printf("Warning: Failed to count documents for policy %s: %v", policy.DocumentType, err)
			continue
		}

		// Count expired documents
		var expired int64
		if err := rm.db.Model(&GoBDRecord{}).
			Where("document_type = ? AND retention_date < ?", policy.DocumentType, time.Now()).
			Count(&expired).Error; err != nil {
			log.Printf("Warning: Failed to count expired documents for policy %s: %v", policy.DocumentType, err)
			continue
		}

		// Count expiring soon (next 30 days)
		var expiringSoon int64
		thirtyDaysFromNow := time.Now().AddDate(0, 0, 30)
		if err := rm.db.Model(&GoBDRecord{}).
			Where("document_type = ? AND retention_date BETWEEN ? AND ?", 
				policy.DocumentType, time.Now(), thirtyDaysFromNow).
			Count(&expiringSoon).Error; err != nil {
			log.Printf("Warning: Failed to count expiring documents for policy %s: %v", policy.DocumentType, err)
			continue
		}

		status.Policies[policy.DocumentType] = RetentionPolicyStatus{
			Policy:        policy,
			TotalRecords:  total,
			ExpiredCount:  expired,
			ExpiringSoon:  expiringSoon,
		}

		status.TotalDocuments += total
		status.ExpiredDocuments += expired
		status.ExpiringSoon += expiringSoon
	}

	return status, nil
}

// ValidateRetentionCompliance validates that retention policies are being followed
func (rm *RetentionManager) ValidateRetentionCompliance() (*ComplianceValidation, error) {
	validation := &ComplianceValidation{
		CheckedAt: time.Now(),
		Issues:    make([]ComplianceIssue, 0),
	}

	// Check for documents without proper retention dates
	var recordsWithoutRetention int64
	if err := rm.db.Model(&GoBDRecord{}).
		Where("retention_date IS NULL OR retention_date = ?", time.Time{}).
		Count(&recordsWithoutRetention).Error; err != nil {
		return nil, fmt.Errorf("failed to check records without retention: %w", err)
	}

	if recordsWithoutRetention > 0 {
		validation.Issues = append(validation.Issues, ComplianceIssue{
			Type:        "missing_retention_date",
			Severity:    "high",
			Description: fmt.Sprintf("%d records found without proper retention dates", recordsWithoutRetention),
			Count:       recordsWithoutRetention,
		})
	}

	// Check for expired documents that should have been deleted
	var overRetentionRecords int64
	sixMonthsAgo := time.Now().AddDate(0, -6, 0) // Grace period of 6 months
	if err := rm.db.Model(&GoBDRecord{}).
		Where("retention_date < ?", sixMonthsAgo).
		Count(&overRetentionRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to check over-retention records: %w", err)
	}

	if overRetentionRecords > 0 {
		validation.Issues = append(validation.Issues, ComplianceIssue{
			Type:        "over_retention",
			Severity:    "medium",
			Description: fmt.Sprintf("%d records found that are significantly past their retention date", overRetentionRecords),
			Count:       overRetentionRecords,
		})
	}

	// Check for missing policies
	var documentTypes []string
	if err := rm.db.Model(&GoBDRecord{}).
		Distinct("document_type").
		Pluck("document_type", &documentTypes).Error; err != nil {
		return nil, fmt.Errorf("failed to get document types: %w", err)
	}

	for _, docType := range documentTypes {
		var policyCount int64
		if err := rm.db.Model(&RetentionPolicy{}).
			Where("document_type = ? AND is_active = ?", docType, true).
			Count(&policyCount).Error; err != nil {
			continue
		}

		if policyCount == 0 {
			validation.Issues = append(validation.Issues, ComplianceIssue{
				Type:        "missing_policy",
				Severity:    "high",
				Description: fmt.Sprintf("No retention policy found for document type: %s", docType),
				DocumentType: docType,
			})
		}
	}

	// Determine overall compliance status
	validation.IsCompliant = len(validation.Issues) == 0
	if !validation.IsCompliant {
		highSeverityIssues := 0
		for _, issue := range validation.Issues {
			if issue.Severity == "high" {
				highSeverityIssues++
			}
		}
		if highSeverityIssues > 0 {
			validation.ComplianceLevel = "non_compliant"
		} else {
			validation.ComplianceLevel = "partially_compliant"
		}
	} else {
		validation.ComplianceLevel = "fully_compliant"
	}

	return validation, nil
}

// RetentionCleanupReport represents the result of a retention cleanup operation
type RetentionCleanupReport struct {
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	Duration           time.Duration `json:"duration"`
	TotalExpired       int           `json:"total_expired"`
	MarkedForDeletion  int           `json:"marked_for_deletion"`
	SkippedAutoDelete  int           `json:"skipped_auto_delete"`
	Errors             []string      `json:"errors"`
}

// RetentionStatus provides an overview of retention status
type RetentionStatus struct {
	TotalDocuments   int64                           `json:"total_documents"`
	ExpiredDocuments int64                           `json:"expired_documents"`
	ExpiringSoon     int64                           `json:"expiring_soon"`
	Policies         map[string]RetentionPolicyStatus `json:"policies"`
}

// RetentionPolicyStatus provides status for a specific retention policy
type RetentionPolicyStatus struct {
	Policy       RetentionPolicy `json:"policy"`
	TotalRecords int64           `json:"total_records"`
	ExpiredCount int64           `json:"expired_count"`
	ExpiringSoon int64           `json:"expiring_soon"`
}

// ComplianceValidation represents the result of compliance validation
type ComplianceValidation struct {
	CheckedAt       time.Time        `json:"checked_at"`
	IsCompliant     bool             `json:"is_compliant"`
	ComplianceLevel string           `json:"compliance_level"` // fully_compliant, partially_compliant, non_compliant
	Issues          []ComplianceIssue `json:"issues"`
}

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	Type         string `json:"type"`         // missing_retention_date, over_retention, missing_policy
	Severity     string `json:"severity"`     // low, medium, high, critical
	Description  string `json:"description"`
	DocumentType string `json:"document_type,omitempty"`
	Count        int64  `json:"count,omitempty"`
}