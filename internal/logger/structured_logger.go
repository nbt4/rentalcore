package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LogLevel represents logging severity levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	Message      string                 `json:"message"`
	Service      string                 `json:"service"`
	Version      string                 `json:"version"`
	Environment  string                 `json:"environment"`
	RequestID    string                 `json:"request_id,omitempty"`
	UserID       *uint                  `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Duration     string                 `json:"duration,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Stack        string                 `json:"stack,omitempty"`
	Component    string                 `json:"component,omitempty"`
	Operation    string                 `json:"operation,omitempty"`
	Resource     string                 `json:"resource,omitempty"`
	Query        string                 `json:"query,omitempty"`
	QueryTime    string                 `json:"query_time,omitempty"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	File         string                 `json:"file,omitempty"`
	Line         int                    `json:"line,omitempty"`
	Function     string                 `json:"function,omitempty"`
}

// StructuredLogger provides production-ready logging
type StructuredLogger struct {
	level       LogLevel
	service     string
	version     string
	environment string
	output      *os.File
	enableCaller bool
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level        LogLevel
	Service      string
	Version      string
	Environment  string
	OutputPath   string
	EnableCaller bool
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	var output *os.File
	var err error

	if config.OutputPath == "" || config.OutputPath == "stdout" {
		output = os.Stdout
	} else {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		output, err = os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	return &StructuredLogger{
		level:        config.Level,
		service:      config.Service,
		version:      config.Version,
		environment:  config.Environment,
		output:       output,
		enableCaller: config.EnableCaller,
	}, nil
}

// log writes a structured log entry
func (sl *StructuredLogger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < sl.level {
		return
	}

	entry := &LogEntry{
		Timestamp:   time.Now().UTC(),
		Level:       level.String(),
		Message:     message,
		Service:     sl.service,
		Version:     sl.version,
		Environment: sl.environment,
		Fields:      fields,
	}

	// Add caller information if enabled
	if sl.enableCaller {
		if file, line, fn := sl.getCaller(3); file != "" {
			entry.File = file
			entry.Line = line
			entry.Function = fn
		}
	}

	// Write JSON log entry
	jsonData, _ := json.Marshal(entry)
	fmt.Fprintf(sl.output, "%s\n", jsonData)
}

// Debug logs debug messages
func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	sl.log(DEBUG, message, sl.mergeFields(fields...))
}

// Info logs info messages
func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	sl.log(INFO, message, sl.mergeFields(fields...))
}

// Warn logs warning messages
func (sl *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	sl.log(WARN, message, sl.mergeFields(fields...))
}

// Error logs error messages
func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	logFields := sl.mergeFields(fields...)
	if err != nil {
		logFields["error"] = err.Error()
		logFields["stack"] = sl.getStackTrace()
	}
	sl.log(ERROR, message, logFields)
}

// Fatal logs fatal messages and exits
func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	logFields := sl.mergeFields(fields...)
	if err != nil {
		logFields["error"] = err.Error()
		logFields["stack"] = sl.getStackTrace()
	}
	sl.log(FATAL, message, logFields)
	os.Exit(1)
}

// LogRequest logs HTTP request details
func (sl *StructuredLogger) LogRequest(c *gin.Context, duration time.Duration, fields ...map[string]interface{}) {
	logFields := sl.mergeFields(fields...)
	
	entry := &LogEntry{
		Timestamp:   time.Now().UTC(),
		Level:       INFO.String(),
		Message:     "HTTP Request",
		Service:     sl.service,
		Version:     sl.version,
		Environment: sl.environment,
		RequestID:   sl.getRequestID(c),
		Method:      c.Request.Method,
		Path:        c.Request.URL.Path,
		StatusCode:  c.Writer.Status(),
		Duration:    duration.String(),
		IP:          c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fields:      logFields,
	}

	// Add user information if available
	if user, exists := c.Get("user"); exists {
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, ok := userMap["id"].(uint); ok {
				entry.UserID = &id
			}
			if username, ok := userMap["username"].(string); ok {
				entry.Username = username
			}
		}
	}

	jsonData, _ := json.Marshal(entry)
	fmt.Fprintf(sl.output, "%s\n", jsonData)
}

// LogQuery logs database query details
func (sl *StructuredLogger) LogQuery(query string, duration time.Duration, err error, fields ...map[string]interface{}) {
	level := INFO
	message := "Database Query"
	
	logFields := sl.mergeFields(fields...)
	logFields["query"] = query
	logFields["query_time"] = duration.String()

	if err != nil {
		level = ERROR
		message = "Database Query Error"
		logFields["error"] = err.Error()
	} else if duration > 500*time.Millisecond {
		level = WARN
		message = "Slow Database Query"
	}

	sl.log(level, message, logFields)
}

// LogBusinessEvent logs business-specific events
func (sl *StructuredLogger) LogBusinessEvent(event string, resource string, operation string, fields ...map[string]interface{}) {
	logFields := sl.mergeFields(fields...)
	logFields["component"] = "business"
	logFields["operation"] = operation
	logFields["resource"] = resource

	sl.log(INFO, event, logFields)
}

// LogSecurityEvent logs security-related events
func (sl *StructuredLogger) LogSecurityEvent(event string, severity string, fields ...map[string]interface{}) {
	level := INFO
	switch severity {
	case "high":
		level = ERROR
	case "medium":
		level = WARN
	case "low":
		level = INFO
	}

	logFields := sl.mergeFields(fields...)
	logFields["component"] = "security"
	logFields["severity"] = severity

	sl.log(level, event, logFields)
}

// LogSystemEvent logs system-level events
func (sl *StructuredLogger) LogSystemEvent(event string, fields ...map[string]interface{}) {
	logFields := sl.mergeFields(fields...)
	logFields["component"] = "system"

	sl.log(INFO, event, logFields)
}

// WithContext returns a context-aware logger
func (sl *StructuredLogger) WithContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		logger: sl,
		ctx:    ctx,
	}
}

// WithRequestContext returns a request-aware logger
func (sl *StructuredLogger) WithRequestContext(c *gin.Context) *RequestLogger {
	return &RequestLogger{
		logger: sl,
		ctx:    c,
	}
}

// getCaller returns caller information
func (sl *StructuredLogger) getCaller(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, ""
	}

	fn := runtime.FuncForPC(pc)
	var fnName string
	if fn != nil {
		fnName = fn.Name()
		// Extract just the function name
		if parts := strings.Split(fnName, "."); len(parts) > 0 {
			fnName = parts[len(parts)-1]
		}
	}

	// Extract just the filename
	if parts := strings.Split(file, "/"); len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return file, line, fnName
}

// getStackTrace returns formatted stack trace
func (sl *StructuredLogger) getStackTrace() string {
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)
	return string(stack[:length])
}

// mergeFields merges multiple field maps
func (sl *StructuredLogger) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range fields {
		for k, v := range field {
			result[k] = v
		}
	}
	return result
}

// getRequestID extracts or generates request ID
func (sl *StructuredLogger) getRequestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-ID"); id != "" {
		return id
	}
	if id := c.GetString("request_id"); id != "" {
		return id
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ContextLogger provides context-aware logging
type ContextLogger struct {
	logger *StructuredLogger
	ctx    context.Context
}

// Debug logs debug with context
func (cl *ContextLogger) Debug(message string, fields ...map[string]interface{}) {
	cl.logger.Debug(message, fields...)
}

// Info logs info with context
func (cl *ContextLogger) Info(message string, fields ...map[string]interface{}) {
	cl.logger.Info(message, fields...)
}

// Warn logs warning with context
func (cl *ContextLogger) Warn(message string, fields ...map[string]interface{}) {
	cl.logger.Warn(message, fields...)
}

// Error logs error with context
func (cl *ContextLogger) Error(message string, err error, fields ...map[string]interface{}) {
	cl.logger.Error(message, err, fields...)
}

// RequestLogger provides request-aware logging
type RequestLogger struct {
	logger *StructuredLogger
	ctx    *gin.Context
}

// Debug logs debug with request context
func (rl *RequestLogger) Debug(message string, fields ...map[string]interface{}) {
	enrichedFields := rl.enrichWithRequestContext(fields...)
	rl.logger.Debug(message, enrichedFields)
}

// Info logs info with request context
func (rl *RequestLogger) Info(message string, fields ...map[string]interface{}) {
	enrichedFields := rl.enrichWithRequestContext(fields...)
	rl.logger.Info(message, enrichedFields)
}

// Warn logs warning with request context
func (rl *RequestLogger) Warn(message string, fields ...map[string]interface{}) {
	enrichedFields := rl.enrichWithRequestContext(fields...)
	rl.logger.Warn(message, enrichedFields)
}

// Error logs error with request context
func (rl *RequestLogger) Error(message string, err error, fields ...map[string]interface{}) {
	enrichedFields := rl.enrichWithRequestContext(fields...)
	rl.logger.Error(message, err, enrichedFields)
}

// enrichWithRequestContext adds request context to fields
func (rl *RequestLogger) enrichWithRequestContext(fields ...map[string]interface{}) map[string]interface{} {
	enriched := rl.logger.mergeFields(fields...)
	enriched["request_id"] = rl.logger.getRequestID(rl.ctx)
	enriched["method"] = rl.ctx.Request.Method
	enriched["path"] = rl.ctx.Request.URL.Path
	enriched["ip"] = rl.ctx.ClientIP()
	return enriched
}

// LoggingMiddleware provides request logging middleware
func (sl *StructuredLogger) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Skip logging for health checks and static files
		if path == "/health" || strings.HasPrefix(path, "/static/") {
			c.Next()
			return
		}

		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", start.UnixNano())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Build full path
		if raw != "" {
			path = path + "?" + raw
		}

		// Log request
		fields := map[string]interface{}{
			"bytes_in":  c.Request.ContentLength,
			"bytes_out": c.Writer.Size(),
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		sl.LogRequest(c, duration, fields)
	}
}

// Close closes the logger output
func (sl *StructuredLogger) Close() error {
	if sl.output != os.Stdout && sl.output != os.Stderr {
		return sl.output.Close()
	}
	return nil
}

// Global logger instance
var GlobalLogger *StructuredLogger

// InitializeLogger initializes the global logger
func InitializeLogger(config LoggerConfig) error {
	var err error
	GlobalLogger, err = NewStructuredLogger(config)
	return err
}