package handlers

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"go-barcode-webapp/internal/cache"
	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/middleware"
	"go-barcode-webapp/internal/monitoring"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MonitoringHandler provides monitoring and dashboard endpoints
type MonitoringHandler struct {
	db           *gorm.DB
	errorTracker *monitoring.ErrorTracker
	perfMonitor  *middleware.PerformanceMonitor
	cache        *cache.CacheManager
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(
	db *gorm.DB,
	errorTracker *monitoring.ErrorTracker,
	perfMonitor *middleware.PerformanceMonitor,
	cache *cache.CacheManager,
) *MonitoringHandler {
	return &MonitoringHandler{
		db:           db,
		errorTracker: errorTracker,
		perfMonitor:  perfMonitor,
		cache:        cache,
	}
}

// Dashboard displays the monitoring dashboard
func (h *MonitoringHandler) Dashboard(c *gin.Context) {
	log.Printf("⚠️ MONITORING DASHBOARD HANDLER CALLED - URL: %s", c.Request.URL.Path)
	user, exists := GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Check if user has monitoring permissions
	if !h.hasMonitoringPermission(user) {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"error": "Access denied: Monitoring dashboard requires admin privileges",
			"user":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "monitoring_dashboard_standalone.html", gin.H{
		"title": "System Monitoring Dashboard",
		"user":  user,
	})
}

// GetSystemMetrics returns system metrics in JSON format
func (h *MonitoringHandler) GetSystemMetrics(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get performance metrics
	perfMetrics := h.perfMonitor.GetMetrics()

	// Get database stats
	dbStats, err := config.GetDatabaseStats(h.db)
	if err != nil {
		dbStats = map[string]interface{}{"error": "Unable to fetch database stats"}
	}

	// Get cache stats
	cacheStats := h.cache.GetAllStats()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get error summary
	errorSummary := h.errorTracker.GetErrorSummary()

	// Get top slow endpoints
	slowEndpoints := h.perfMonitor.GetTopSlowEndpoints(10)

	// Calculate uptime
	uptime := time.Since(time.Now().Add(-time.Duration(perfMetrics.RequestCount) * time.Second)).String()

	metrics := gin.H{
		"timestamp": time.Now().UTC(),
		"system": gin.H{
			"uptime":           uptime,
			"goroutines":       runtime.NumGoroutine(),
			"cpu_cores":        runtime.NumCPU(),
			"go_version":       runtime.Version(),
		},
		"performance": gin.H{
			"request_count":    perfMetrics.RequestCount,
			"average_response": perfMetrics.AverageResponse.String(),
			"error_rate":       perfMetrics.ErrorRate,
			"slow_endpoints":   slowEndpoints,
		},
		"memory": gin.H{
			"allocated":      formatBytes(memStats.Alloc),
			"total_alloc":    formatBytes(memStats.TotalAlloc),
			"sys":            formatBytes(memStats.Sys),
			"heap_in_use":    formatBytes(memStats.HeapInuse),
			"heap_released":  formatBytes(memStats.HeapReleased),
			"gc_runs":        memStats.NumGC,
			"gc_pause":       memStats.PauseNs[(memStats.NumGC+255)%256],
		},
		"database":     dbStats,
		"cache":        cacheStats,
		"errors":       errorSummary,
	}

	c.JSON(http.StatusOK, metrics)
}

// GetErrorDetails returns detailed error information
func (h *MonitoringHandler) GetErrorDetails(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse query parameters
	resolved := c.Query("resolved") == "true"
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	errors := h.errorTracker.GetErrors(resolved, limit)

	c.JSON(http.StatusOK, gin.H{
		"errors":   errors,
		"resolved": resolved,
		"count":    len(errors),
	})
}

// ResolveError marks an error as resolved
func (h *MonitoringHandler) ResolveError(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error fingerprint is required"})
		return
	}

	err := h.errorTracker.ResolveError(fingerprint, user.Username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Error marked as resolved",
		"fingerprint": fingerprint,
		"resolved_by": user.Username,
	})
}

// GetApplicationHealth returns comprehensive health check
func (h *MonitoringHandler) GetApplicationHealth(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0", // Should come from build info
		"checks":    gin.H{},
	}

	// Database health check
	sqlDB, err := h.db.DB()
	if err != nil {
		health["checks"].(gin.H)["database"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		if err := sqlDB.Ping(); err != nil {
			health["checks"].(gin.H)["database"] = gin.H{
				"status": "unhealthy",
				"error":  "Database ping failed: " + err.Error(),
			}
			health["status"] = "degraded"
		} else {
			health["checks"].(gin.H)["database"] = gin.H{
				"status":     "healthy",
				"connection": "active",
			}
		}
	}

	// Cache health check
	cacheStats := h.cache.GetAllStats()
	health["checks"].(gin.H)["cache"] = gin.H{
		"status": "healthy",
		"stats":  cacheStats,
	}

	// Memory health check
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	memoryStatus := "healthy"
	if memStats.Alloc > 500*1024*1024 { // 500MB threshold
		memoryStatus = "warning"
	}
	if memStats.Alloc > 1024*1024*1024 { // 1GB threshold
		memoryStatus = "critical"
		health["status"] = "degraded"
	}

	health["checks"].(gin.H)["memory"] = gin.H{
		"status":    memoryStatus,
		"allocated": formatBytes(memStats.Alloc),
		"sys":       formatBytes(memStats.Sys),
	}

	// Error rate health check
	perfMetrics := h.perfMonitor.GetMetrics()
	errorStatus := "healthy"
	if perfMetrics.ErrorRate > 5 {
		errorStatus = "warning"
	}
	if perfMetrics.ErrorRate > 15 {
		errorStatus = "critical"
		health["status"] = "degraded"
	}

	health["checks"].(gin.H)["error_rate"] = gin.H{
		"status":     errorStatus,
		"error_rate": perfMetrics.ErrorRate,
		"requests":   perfMetrics.RequestCount,
	}

	// Determine overall status
	if health["status"] == "degraded" {
		c.JSON(http.StatusServiceUnavailable, health)
	} else {
		c.JSON(http.StatusOK, health)
	}
}

// GetPerformanceMetrics returns detailed performance metrics
func (h *MonitoringHandler) GetPerformanceMetrics(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	metrics := h.perfMonitor.GetMetrics()
	
	c.JSON(http.StatusOK, gin.H{
		"performance": metrics,
		"endpoints":   h.perfMonitor.GetTopSlowEndpoints(20),
	})
}

// ExportMetrics exports metrics in Prometheus format
func (h *MonitoringHandler) ExportMetrics(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.String(http.StatusUnauthorized, "Authentication required")
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	metrics := h.perfMonitor.GetMetrics()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Generate Prometheus format metrics
	prometheus := `# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total %d

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds %f

# HELP http_error_rate_percent HTTP error rate percentage
# TYPE http_error_rate_percent gauge
http_error_rate_percent %f

# HELP memory_allocated_bytes Currently allocated memory in bytes
# TYPE memory_allocated_bytes gauge
memory_allocated_bytes %d

# HELP memory_sys_bytes Total memory obtained from OS
# TYPE memory_sys_bytes gauge
memory_sys_bytes %d

# HELP goroutines_total Current number of goroutines
# TYPE goroutines_total gauge
goroutines_total %d

# HELP gc_runs_total Total number of GC runs
# TYPE gc_runs_total counter
gc_runs_total %d
`

	output := fmt.Sprintf(prometheus,
		metrics.RequestCount,
		metrics.AverageResponse.Seconds(),
		metrics.ErrorRate,
		memStats.Alloc,
		memStats.Sys,
		runtime.NumGoroutine(),
		memStats.NumGC,
	)

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, output)
}

// GetLogStream streams application logs
func (h *MonitoringHandler) GetLogStream(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// TODO: Implement log streaming
	// This would require a log aggregator or file tailing mechanism
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Log streaming not yet implemented",
	})
}

// TriggerTestError creates a test error for monitoring validation
func (h *MonitoringHandler) TriggerTestError(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if !h.hasMonitoringPermission(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get severity from query parameter
	severityStr := c.DefaultQuery("severity", "medium")
	severity := monitoring.MEDIUM

	switch severityStr {
	case "low":
		severity = monitoring.LOW
	case "medium":
		severity = monitoring.MEDIUM
	case "high":
		severity = monitoring.HIGH
	case "critical":
		severity = monitoring.CRITICAL
	}

	// Create test error
	h.errorTracker.CaptureRequestError(c, "Test Error", 
		fmt.Errorf("This is a test error triggered by %s", user.Username), 
		severity,
		map[string]interface{}{
			"test":        true,
			"triggered_by": user.Username,
		})

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Test error created",
		"severity": severityStr,
	})
}

// hasMonitoringPermission checks if user has monitoring permissions
func (h *MonitoringHandler) hasMonitoringPermission(user interface{}) bool {
	// TODO: Implement proper role-based access control
	// For now, allow all authenticated users
	return true
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}