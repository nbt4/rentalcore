package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// AuditLogger provides GoBD-compliant audit logging
type AuditLogger struct {
	db          *gorm.DB
	lastHash    string
	integrityMu sync.RWMutex
}

// AuditStatistics provides audit logging statistics
type AuditStatistics struct {
	TotalEvents     int64 `json:"total_events"`
	EventsByType    map[string]int64 `json:"events_by_type"`
	EventsByUser    map[string]int64 `json:"events_by_user"`
	IntegrityChecks int64 `json:"integrity_checks"`
	LastEvent       time.Time `json:"last_event"`
	ChainIntact     bool  `json:"chain_intact"`
}

// NewAuditLogger creates a new GoBD-compliant audit logger
func NewAuditLogger(db *gorm.DB) (*AuditLogger, error) {
	al := &AuditLogger{
		db: db,
	}

	// Get the last hash from the chain to maintain integrity
	var lastEvent AuditEvent
	if err := db.Order("timestamp DESC").First(&lastEvent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// First event, no previous hash
			al.lastHash = ""
		} else {
			return nil, fmt.Errorf("failed to get last audit event: %w", err)
		}
	} else {
		al.lastHash = lastEvent.EventHash
	}

	return al, nil
}

// LogEvent logs an audit event with GoBD compliance
func (al *AuditLogger) LogEvent(eventType, objectType, objectID string, userID uint, action string, oldValues, newValues interface{}) error {
	return al.LogEventWithContext(eventType, objectType, objectID, userID, "", action, oldValues, newValues, "", "", "", make(map[string]interface{}))
}

// LogEventWithContext logs an audit event with full context information
func (al *AuditLogger) LogEventWithContext(
	eventType, objectType, objectID string,
	userID uint, username, action string,
	oldValues, newValues interface{},
	ipAddress, userAgent, sessionID string,
	context map[string]interface{},
) error {
	al.integrityMu.Lock()
	defer al.integrityMu.Unlock()

	// Serialize values
	var oldValuesJSON, newValuesJSON string
	if oldValues != nil {
		if data, err := json.Marshal(oldValues); err == nil {
			oldValuesJSON = string(data)
		}
	}
	if newValues != nil {
		if data, err := json.Marshal(newValues); err == nil {
			newValuesJSON = string(data)
		}
	}

	// Get username if not provided
	if username == "" {
		var user struct {
			Username string
		}
		if err := al.db.Table("users").Select("username").Where("userID = ?", userID).First(&user).Error; err == nil {
			username = user.Username
		} else {
			username = fmt.Sprintf("user_%d", userID)
		}
	}

	// Calculate retention date (10 years for audit logs according to GoBD)
	retentionDate := time.Now().AddDate(10, 0, 0)

	// Create audit event
	event := &AuditEvent{
		EventType:     eventType,
		ObjectType:    objectType,
		ObjectID:      objectID,
		UserID:        userID,
		Username:      username,
		Action:        action,
		OldValues:     oldValuesJSON,
		NewValues:     newValuesJSON,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		SessionID:     sessionID,
		Context:       context,
		PreviousHash:  al.lastHash,
		IsCompliant:   true,
		RetentionDate: retentionDate,
		Timestamp:     time.Now(),
	}

	// Generate event hash for integrity chain
	event.EventHash = al.generateEventHash(event)

	// Save to database
	if err := al.db.Create(event).Error; err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	// Update last hash for chain integrity
	al.lastHash = event.EventHash

	return nil
}

// LogInvoiceEvent logs invoice-specific audit events
func (al *AuditLogger) LogInvoiceEvent(eventType string, invoiceID uint, userID uint, action string, oldData, newData interface{}, ipAddress, userAgent, sessionID string) error {
	context := map[string]interface{}{
		"document_type": "invoice",
		"gobd_relevant": true,
		"tax_relevant":  true,
	}

	return al.LogEventWithContext(
		eventType, "invoice", fmt.Sprintf("%d", invoiceID),
		userID, "", action,
		oldData, newData,
		ipAddress, userAgent, sessionID,
		context,
	)
}

// LogCustomerEvent logs customer data audit events (GDPR relevant)
func (al *AuditLogger) LogCustomerEvent(eventType string, customerID uint, userID uint, action string, oldData, newData interface{}, ipAddress, userAgent, sessionID string) error {
	context := map[string]interface{}{
		"document_type": "customer_data",
		"gdpr_relevant": true,
		"pii_involved":  true,
	}

	return al.LogEventWithContext(
		eventType, "customer", fmt.Sprintf("%d", customerID),
		userID, "", action,
		oldData, newData,
		ipAddress, userAgent, sessionID,
		context,
	)
}

// LogSystemEvent logs system-level events
func (al *AuditLogger) LogSystemEvent(eventType, action string, userID uint, context map[string]interface{}, ipAddress, userAgent, sessionID string) error {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["system_event"] = true

	return al.LogEventWithContext(
		eventType, "system", "system",
		userID, "", action,
		nil, nil,
		ipAddress, userAgent, sessionID,
		context,
	)
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(eventType, action string, userID uint, severity string, context map[string]interface{}, ipAddress, userAgent, sessionID string) error {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["security_event"] = true
	context["severity"] = severity
	context["requires_review"] = severity == "high" || severity == "critical"

	return al.LogEventWithContext(
		eventType, "security", "security",
		userID, "", action,
		nil, nil,
		ipAddress, userAgent, sessionID,
		context,
	)
}

// VerifyChainIntegrity verifies the integrity of the entire audit chain
func (al *AuditLogger) VerifyChainIntegrity() (bool, error) {
	var events []AuditEvent
	if err := al.db.Order("timestamp ASC").Find(&events).Error; err != nil {
		return false, fmt.Errorf("failed to retrieve audit events: %w", err)
	}

	if len(events) == 0 {
		return true, nil // Empty chain is valid
	}

	// Verify first event has no previous hash
	if events[0].PreviousHash != "" {
		return false, fmt.Errorf("first event should not have previous hash")
	}

	// Verify chain integrity
	for i := 1; i < len(events); i++ {
		expectedPrevious := events[i-1].EventHash
		if events[i].PreviousHash != expectedPrevious {
			return false, fmt.Errorf("chain integrity broken at event %d", events[i].ID)
		}

		// Verify event hash
		expectedHash := al.generateEventHash(&events[i])
		if events[i].EventHash != expectedHash {
			return false, fmt.Errorf("event hash verification failed for event %d", events[i].ID)
		}
	}

	return true, nil
}

// GetAuditTrail gets the audit trail for a specific object
func (al *AuditLogger) GetAuditTrail(objectType, objectID string) ([]AuditEvent, error) {
	var events []AuditEvent
	if err := al.db.Where("object_type = ? AND object_id = ?", objectType, objectID).
		Order("timestamp ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to get audit trail: %w", err)
	}
	return events, nil
}

// GetAuditEvents gets audit events with filtering and pagination
func (al *AuditLogger) GetAuditEvents(filters AuditFilters) ([]AuditEvent, int64, error) {
	query := al.db.Model(&AuditEvent{})

	// Apply filters
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.ObjectType != "" {
		query = query.Where("object_type = ?", filters.ObjectType)
	}
	if filters.UserID > 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if !filters.StartDate.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartDate)
	}
	if !filters.EndDate.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndDate)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Apply pagination and sorting
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	query = query.Order("timestamp DESC")

	var events []AuditEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit events: %w", err)
	}

	return events, total, nil
}

// GetStatistics returns audit logging statistics
func (al *AuditLogger) GetStatistics() (*AuditStatistics, error) {
	stats := &AuditStatistics{
		EventsByType: make(map[string]int64),
		EventsByUser: make(map[string]int64),
	}

	// Total events
	if err := al.db.Model(&AuditEvent{}).Count(&stats.TotalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count total events: %w", err)
	}

	// Events by type
	var typeResults []struct {
		EventType string
		Count     int64
	}
	if err := al.db.Model(&AuditEvent{}).
		Select("event_type, COUNT(*) as count").
		Group("event_type").
		Find(&typeResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	for _, result := range typeResults {
		stats.EventsByType[result.EventType] = result.Count
	}

	// Events by user
	var userResults []struct {
		Username string
		Count    int64
	}
	if err := al.db.Model(&AuditEvent{}).
		Select("username, COUNT(*) as count").
		Group("username").
		Find(&userResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get events by user: %w", err)
	}
	for _, result := range userResults {
		stats.EventsByUser[result.Username] = result.Count
	}

	// Last event
	var lastEvent AuditEvent
	if err := al.db.Order("timestamp DESC").First(&lastEvent).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get last event: %w", err)
		}
	} else {
		stats.LastEvent = lastEvent.Timestamp
	}

	// Verify chain integrity
	intact, err := al.VerifyChainIntegrity()
	if err != nil {
		return nil, fmt.Errorf("failed to verify chain integrity: %w", err)
	}
	stats.ChainIntact = intact
	stats.IntegrityChecks++ // Increment check counter

	return stats, nil
}

// generateEventHash generates a hash for audit event integrity
func (al *AuditLogger) generateEventHash(event *AuditEvent) string {
	hashData := fmt.Sprintf("%s:%s:%s:%d:%s:%s:%s:%s:%s",
		event.EventType,
		event.ObjectType,
		event.ObjectID,
		event.UserID,
		event.Action,
		event.PreviousHash,
		event.Timestamp.Format(time.RFC3339Nano),
		event.OldValues,
		event.NewValues,
	)
	
	hash := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(hash[:])
}

// AuditFilters defines filters for audit event queries
type AuditFilters struct {
	EventType  string    `form:"event_type"`
	ObjectType string    `form:"object_type"`
	UserID     uint      `form:"user_id"`
	StartDate  time.Time `form:"start_date"`
	EndDate    time.Time `form:"end_date"`
	Limit      int       `form:"limit"`
	Offset     int       `form:"offset"`
}

