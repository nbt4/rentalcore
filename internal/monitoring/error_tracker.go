package monitoring

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorSeverity represents error severity levels
type ErrorSeverity int

const (
	LOW ErrorSeverity = iota
	MEDIUM
	HIGH
	CRITICAL
)

// String returns string representation of error severity
func (es ErrorSeverity) String() string {
	switch es {
	case LOW:
		return "LOW"
	case MEDIUM:
		return "MEDIUM"
	case HIGH:
		return "HIGH"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorDetails represents detailed error information
type ErrorDetails struct {
	ID          string                 `json:"id"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error"`
	Severity    string                 `json:"severity"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      *uint                  `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	IP          string                 `json:"ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Stack       []StackFrame           `json:"stack,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Fingerprint string                 `json:"fingerprint"`
	Count       int                    `json:"count"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Resolved    bool                   `json:"resolved"`
	ResolvedBy  string                 `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// StackFrame represents a single stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// ErrorSummary represents error summary for reporting
type ErrorSummary struct {
	Count       int       `json:"count"`
	LastOccured time.Time `json:"last_occured"`
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	Message     string    `json:"message"`
}

// AlertRule defines when to send alerts
type AlertRule struct {
	Severity    ErrorSeverity
	Threshold   int           // Number of occurrences
	TimeWindow  time.Duration // Time window for threshold
	Component   string        // Optional component filter
	Enabled     bool
}

// AlertChannel defines how alerts are sent
type AlertChannel interface {
	SendAlert(error *ErrorDetails, rule *AlertRule) error
}

// EmailAlertChannel sends alerts via email
type EmailAlertChannel struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromEmail    string
	ToEmails     []string
	TemplatePath string
}

// SendAlert sends email alert
func (eac *EmailAlertChannel) SendAlert(error *ErrorDetails, rule *AlertRule) error {
	// TODO: Implement email sending logic
	fmt.Printf("EMAIL ALERT: %s - %s\n", error.Severity, error.Message)
	return nil
}

// SlackAlertChannel sends alerts to Slack
type SlackAlertChannel struct {
	WebhookURL string
	Channel    string
	Username   string
}

// SendAlert sends Slack alert
func (sac *SlackAlertChannel) SendAlert(error *ErrorDetails, rule *AlertRule) error {
	// TODO: Implement Slack webhook logic
	fmt.Printf("SLACK ALERT: %s - %s\n", error.Severity, error.Message)
	return nil
}

// ErrorTracker manages error tracking and alerting
type ErrorTracker struct {
	errors       map[string]*ErrorDetails
	rules        []AlertRule
	channels     []AlertChannel
	mutex        sync.RWMutex
	maxErrors    int
	retention    time.Duration
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(maxErrors int, retention time.Duration) *ErrorTracker {
	tracker := &ErrorTracker{
		errors:    make(map[string]*ErrorDetails),
		rules:     make([]AlertRule, 0),
		channels:  make([]AlertChannel, 0),
		maxErrors: maxErrors,
		retention: retention,
	}

	// Start cleanup goroutine
	go tracker.cleanup()

	return tracker
}

// CaptureError captures and processes an error
func (et *ErrorTracker) CaptureError(message string, err error, severity ErrorSeverity, context map[string]interface{}) *ErrorDetails {
	errorDetails := &ErrorDetails{
		ID:          et.generateErrorID(),
		Message:     message,
		Error:       "",
		Severity:    severity.String(),
		Timestamp:   time.Now().UTC(),
		Context:     context,
		Count:       1,
		FirstSeen:   time.Now().UTC(),
		LastSeen:    time.Now().UTC(),
		Resolved:    false,
	}

	if err != nil {
		errorDetails.Error = err.Error()
	}

	// Extract component and operation from context
	if comp, exists := context["component"]; exists {
		if component, ok := comp.(string); ok {
			errorDetails.Component = component
		}
	}
	if op, exists := context["operation"]; exists {
		if operation, ok := op.(string); ok {
			errorDetails.Operation = operation
		}
	}

	// Generate stack trace
	errorDetails.Stack = et.getStackTrace(3) // Skip 3 frames to get to caller

	// Generate fingerprint for deduplication
	errorDetails.Fingerprint = et.generateFingerprint(errorDetails)

	// Store or update error
	et.storeError(errorDetails)

	// Check alert rules
	et.checkAlertRules(errorDetails)

	return errorDetails
}

// CaptureRequestError captures an error from HTTP request context
func (et *ErrorTracker) CaptureRequestError(c *gin.Context, message string, err error, severity ErrorSeverity, context map[string]interface{}) *ErrorDetails {
	if context == nil {
		context = make(map[string]interface{})
	}

	// Add request context
	context["method"] = c.Request.Method
	context["path"] = c.Request.URL.Path
	context["ip"] = c.ClientIP()
	context["user_agent"] = c.GetHeader("User-Agent")
	context["request_id"] = c.GetString("request_id")

	// Add user context if available
	if user, exists := c.Get("user"); exists {
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, ok := userMap["id"].(uint); ok {
				context["user_id"] = id
			}
			if username, ok := userMap["username"].(string); ok {
				context["username"] = username
			}
		}
	}

	errorDetails := et.CaptureError(message, err, severity, context)

	// Update with request-specific fields
	et.mutex.Lock()
	if stored, exists := et.errors[errorDetails.Fingerprint]; exists {
		stored.Method = c.Request.Method
		stored.Path = c.Request.URL.Path
		stored.IP = c.ClientIP()
		stored.UserAgent = c.GetHeader("User-Agent")
		stored.RequestID = c.GetString("request_id")

		if user, exists := c.Get("user"); exists {
			if userMap, ok := user.(map[string]interface{}); ok {
				if id, ok := userMap["id"].(uint); ok {
					stored.UserID = &id
				}
				if username, ok := userMap["username"].(string); ok {
					stored.Username = username
				}
			}
		}
	}
	et.mutex.Unlock()

	return errorDetails
}

// CaptureBusinessError captures business logic errors
func (et *ErrorTracker) CaptureBusinessError(component, operation, message string, err error, severity ErrorSeverity, context map[string]interface{}) *ErrorDetails {
	if context == nil {
		context = make(map[string]interface{})
	}

	context["component"] = component
	context["operation"] = operation
	context["type"] = "business"

	return et.CaptureError(message, err, severity, context)
}

// CaptureSystemError captures system-level errors
func (et *ErrorTracker) CaptureSystemError(component, message string, err error, severity ErrorSeverity, context map[string]interface{}) *ErrorDetails {
	if context == nil {
		context = make(map[string]interface{})
	}

	context["component"] = component
	context["type"] = "system"

	return et.CaptureError(message, err, severity, context)
}

// GetErrors returns all tracked errors
func (et *ErrorTracker) GetErrors(resolved bool, limit int) []*ErrorDetails {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	var errors []*ErrorDetails
	for _, error := range et.errors {
		if error.Resolved == resolved {
			errors = append(errors, error)
		}
	}

	// Sort by last seen (most recent first)
	for i := 0; i < len(errors)-1; i++ {
		for j := 0; j < len(errors)-i-1; j++ {
			if errors[j].LastSeen.Before(errors[j+1].LastSeen) {
				errors[j], errors[j+1] = errors[j+1], errors[j]
			}
		}
	}

	if limit > 0 && limit < len(errors) {
		errors = errors[:limit]
	}

	return errors
}

// GetErrorSummary returns error summary by component
func (et *ErrorTracker) GetErrorSummary() map[string]ErrorSummary {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	summary := make(map[string]ErrorSummary)

	for _, error := range et.errors {
		if error.Resolved {
			continue
		}

		key := error.Component
		if key == "" {
			key = "unknown"
		}

		if existing, exists := summary[key]; exists {
			existing.Count += error.Count
			if error.LastSeen.After(existing.LastOccured) {
				existing.LastOccured = error.LastSeen
			}
			summary[key] = existing
		} else {
			summary[key] = ErrorSummary{
				Count:       error.Count,
				LastOccured: error.LastSeen,
				Severity:    error.Severity,
				Component:   error.Component,
				Message:     error.Message,
			}
		}
	}

	return summary
}

// ResolveError marks an error as resolved
func (et *ErrorTracker) ResolveError(fingerprint, resolvedBy string) error {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	if error, exists := et.errors[fingerprint]; exists {
		now := time.Now().UTC()
		error.Resolved = true
		error.ResolvedBy = resolvedBy
		error.ResolvedAt = &now
		return nil
	}

	return fmt.Errorf("error not found: %s", fingerprint)
}

// AddAlertRule adds an alert rule
func (et *ErrorTracker) AddAlertRule(rule AlertRule) {
	et.mutex.Lock()
	defer et.mutex.Unlock()
	et.rules = append(et.rules, rule)
}

// AddAlertChannel adds an alert channel
func (et *ErrorTracker) AddAlertChannel(channel AlertChannel) {
	et.mutex.Lock()
	defer et.mutex.Unlock()
	et.channels = append(et.channels, channel)
}

// storeError stores or updates an error
func (et *ErrorTracker) storeError(errorDetails *ErrorDetails) {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	if existing, exists := et.errors[errorDetails.Fingerprint]; exists {
		// Update existing error
		existing.Count++
		existing.LastSeen = errorDetails.Timestamp
		existing.Context = errorDetails.Context // Update context with latest
	} else {
		// Store new error
		et.errors[errorDetails.Fingerprint] = errorDetails

		// Enforce maximum errors limit
		if len(et.errors) > et.maxErrors {
			et.evictOldestError()
		}
	}
}

// evictOldestError removes the oldest error
func (et *ErrorTracker) evictOldestError() {
	var oldest *ErrorDetails
	var oldestKey string

	for key, error := range et.errors {
		if oldest == nil || error.FirstSeen.Before(oldest.FirstSeen) {
			oldest = error
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(et.errors, oldestKey)
	}
}

// checkAlertRules checks if any alert rules are triggered
func (et *ErrorTracker) checkAlertRules(errorDetails *ErrorDetails) {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	for _, rule := range et.rules {
		if !rule.Enabled {
			continue
		}

		// Check severity match
		errorSeverity := et.parseSeverity(errorDetails.Severity)
		if errorSeverity < rule.Severity {
			continue
		}

		// Check component match (if specified)
		if rule.Component != "" && rule.Component != errorDetails.Component {
			continue
		}

		// Check threshold within time window
		count := et.countErrorsInWindow(errorDetails.Fingerprint, rule.TimeWindow)
		if count >= rule.Threshold {
			et.sendAlerts(errorDetails, &rule)
		}
	}
}

// countErrorsInWindow counts errors within time window
func (et *ErrorTracker) countErrorsInWindow(fingerprint string, window time.Duration) int {
	if error, exists := et.errors[fingerprint]; exists {
		cutoff := time.Now().UTC().Add(-window)
		if error.LastSeen.After(cutoff) {
			return error.Count
		}
	}
	return 0
}

// sendAlerts sends alerts through all channels
func (et *ErrorTracker) sendAlerts(errorDetails *ErrorDetails, rule *AlertRule) {
	for _, channel := range et.channels {
		go func(ch AlertChannel) {
			if err := ch.SendAlert(errorDetails, rule); err != nil {
				fmt.Printf("Failed to send alert: %v\n", err)
			}
		}(channel)
	}
}

// generateErrorID generates a unique error ID
func (et *ErrorTracker) generateErrorID() string {
	return fmt.Sprintf("err_%d", time.Now().UnixNano())
}

// generateFingerprint generates a fingerprint for error deduplication
func (et *ErrorTracker) generateFingerprint(errorDetails *ErrorDetails) string {
	components := []string{
		errorDetails.Message,
		errorDetails.Component,
		errorDetails.Operation,
	}

	if len(errorDetails.Stack) > 0 {
		// Use top stack frame for fingerprint
		components = append(components, errorDetails.Stack[0].Function, errorDetails.Stack[0].File)
	}

	return fmt.Sprintf("%x", strings.Join(components, "|"))
}

// getStackTrace captures stack trace
func (et *ErrorTracker) getStackTrace(skip int) []StackFrame {
	var frames []StackFrame
	for i := skip; i < skip+10; i++ { // Capture up to 10 frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		funcName := fn.Name()
		packageName := ""
		if parts := strings.Split(funcName, "."); len(parts) > 1 {
			packageName = strings.Join(parts[:len(parts)-1], ".")
			funcName = parts[len(parts)-1]
		}

		// Extract just the filename
		if parts := strings.Split(file, "/"); len(parts) > 0 {
			file = parts[len(parts)-1]
		}

		frames = append(frames, StackFrame{
			Function: funcName,
			File:     file,
			Line:     line,
			Package:  packageName,
		})
	}

	return frames
}

// parseSeverity parses severity string
func (et *ErrorTracker) parseSeverity(severity string) ErrorSeverity {
	switch strings.ToUpper(severity) {
	case "LOW":
		return LOW
	case "MEDIUM":
		return MEDIUM
	case "HIGH":
		return HIGH
	case "CRITICAL":
		return CRITICAL
	default:
		return LOW
	}
}

// cleanup removes old errors
func (et *ErrorTracker) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		et.mutex.Lock()
		cutoff := time.Now().UTC().Add(-et.retention)
		
		for key, error := range et.errors {
			if error.LastSeen.Before(cutoff) && error.Resolved {
				delete(et.errors, key)
			}
		}
		et.mutex.Unlock()
	}
}

// ErrorTrackingMiddleware provides error tracking middleware
func (et *ErrorTracker) ErrorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture panic as critical error
				et.CaptureRequestError(c, "Application Panic", fmt.Errorf("%v", err), CRITICAL, map[string]interface{}{
					"panic": true,
				})

				// Return 500 error
				c.JSON(500, gin.H{"error": "Internal server error"})
				c.Abort()
			}
		}()

		c.Next()

		// Capture errors from context
		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				severity := MEDIUM
				if c.Writer.Status() >= 500 {
					severity = HIGH
				}

				et.CaptureRequestError(c, "Request Error", ginErr.Err, severity, map[string]interface{}{
					"error_type": ginErr.Type,
				})
			}
		}
	}
}

// Global error tracker instance
var GlobalErrorTracker *ErrorTracker

// InitializeErrorTracker initializes the global error tracker
func InitializeErrorTracker(maxErrors int, retention time.Duration) {
	GlobalErrorTracker = NewErrorTracker(maxErrors, retention)

	// Add default alert rules
	GlobalErrorTracker.AddAlertRule(AlertRule{
		Severity:   CRITICAL,
		Threshold:  1,
		TimeWindow: 5 * time.Minute,
		Enabled:    true,
	})

	GlobalErrorTracker.AddAlertRule(AlertRule{
		Severity:   HIGH,
		Threshold:  5,
		TimeWindow: 15 * time.Minute,
		Enabled:    true,
	})

	GlobalErrorTracker.AddAlertRule(AlertRule{
		Severity:   MEDIUM,
		Threshold:  10,
		TimeWindow: 30 * time.Minute,
		Enabled:    true,
	})
}