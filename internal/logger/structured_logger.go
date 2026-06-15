package logger

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
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

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level        LogLevel
	Service      string
	Version      string
	Environment  string
	OutputPath   string
	EnableCaller bool
}

// StructuredLogger wraps zerolog
type StructuredLogger struct {
	logger *zerolog.Logger
	config LoggerConfig
	output *os.File
	level  zerolog.Level
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	var output *os.File

	if config.OutputPath == "" || config.OutputPath == "stdout" {
		output = os.Stdout
	} else {
		var err error
		output, err = os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
	}

	var zlevel zerolog.Level
	switch config.Level {
	case DEBUG:
		zlevel = zerolog.DebugLevel
	case INFO:
		zlevel = zerolog.InfoLevel
	case WARN:
		zlevel = zerolog.WarnLevel
	case ERROR:
		zlevel = zerolog.ErrorLevel
	case FATAL:
		zlevel = zerolog.FatalLevel
	default:
		zlevel = zerolog.InfoLevel
	}

	zl := zerolog.New(output).Level(zlevel).With().Timestamp().
		Str("service", config.Service).
		Str("version", config.Version).
		Str("environment", config.Environment)

	if config.EnableCaller {
		zl = zl.Caller()
	}

	logger := zl.Logger()

	return &StructuredLogger{
		logger: &logger,
		config: config,
		output: output,
		level:  zlevel,
	}, nil
}

func (sl *StructuredLogger) Debug(msg string) {
	sl.logger.Debug().Msg(msg)
}

func (sl *StructuredLogger) Info(msg string) {
	sl.logger.Info().Msg(msg)
}

func (sl *StructuredLogger) Warn(msg string) {
	sl.logger.Warn().Msg(msg)
}

func (sl *StructuredLogger) Error(msg string) {
	sl.logger.Error().Msg(msg)
}

func (sl *StructuredLogger) Fatal(msg string) {
	sl.logger.Fatal().Msg(msg)
}

// Raw returns the underlying zerolog logger
func (sl *StructuredLogger) Raw() *zerolog.Logger {
	return sl.logger
}

// LoggingMiddleware provides request logging middleware
func (sl *StructuredLogger) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Skip logging for health checks
		if path == "/health" {
			c.Next()
			return
		}

		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()

		duration := time.Since(start)

		evt := sl.logger.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.GetHeader("User-Agent")).
			Str("request_id", requestID).
			Int64("bytes_in", c.Request.ContentLength).
			Int("bytes_out", c.Writer.Size())

		if len(c.Errors) > 0 {
			evt.Str("errors", c.Errors.String())
		}

		evt.Msg("HTTP Request")
	}
}

// Close closes the logger output
func (sl *StructuredLogger) Close() error {
	if sl.output != nil && sl.output != os.Stdout && sl.output != os.Stderr {
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

// WithContext returns a context-aware logger
func (sl *StructuredLogger) WithContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		logger: sl,
		ctx:    ctx,
	}
}

// ContextLogger provides context-aware logging
type ContextLogger struct {
	logger *StructuredLogger
	ctx    context.Context
}

func (cl *ContextLogger) Debug(msg string)  { cl.logger.Debug(msg) }
func (cl *ContextLogger) Info(msg string)   { cl.logger.Info(msg) }
func (cl *ContextLogger) Warn(msg string)   { cl.logger.Warn(msg) }
func (cl *ContextLogger) Error(msg string)  { cl.logger.Error(msg) }

// ── Global helper functions for convenient logging ──

// LogInfo logs an info message with formatted string
func LogInfo(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.logger.Info().Msgf(format, args...)
	}
}

// LogWarn logs a warning message with formatted string
func LogWarn(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.logger.Warn().Msgf(format, args...)
	}
}

// LogError logs an error message with formatted string
func LogError(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.logger.Error().Msgf(format, args...)
	}
}

// LogFatal logs a fatal message and exits
func LogFatal(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.logger.Fatal().Msgf(format, args...)
	}
	// Fallback
	panic(fmt.Sprintf(format, args...))
}

// LogDebug logs a debug message with formatted string
func LogDebug(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.logger.Debug().Msgf(format, args...)
	}
}
