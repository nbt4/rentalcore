package middleware

import (
	"compress/gzip"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// PerformanceMetrics stores request performance metrics
type PerformanceMetrics struct {
	RequestCount    int64            `json:"request_count"`
	AverageResponse time.Duration    `json:"average_response"`
	SlowQueries     []SlowQuery      `json:"slow_queries"`
	ErrorRate       float64          `json:"error_rate"`
	MemoryUsage     MemoryStats      `json:"memory_usage"`
	EndpointStats   map[string]Stats `json:"endpoint_stats"`
}

// SlowQuery represents a slow database query
type SlowQuery struct {
	Query     string        `json:"query"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Endpoint  string        `json:"endpoint"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Allocated     uint64 `json:"allocated"`
	TotalAlloc    uint64 `json:"total_alloc"`
	Sys           uint64 `json:"sys"`
	GCRuns        uint32 `json:"gc_runs"`
	HeapInUse     uint64 `json:"heap_in_use"`
	HeapReleased  uint64 `json:"heap_released"`
}

// Stats represents endpoint-specific statistics
type Stats struct {
	Count         int64         `json:"count"`
	TotalDuration time.Duration `json:"total_duration"`
	AverageTime   time.Duration `json:"average_time"`
	ErrorCount    int64         `json:"error_count"`
	SlowCount     int64         `json:"slow_count"`
}

// PerformanceMonitor tracks application performance
type PerformanceMonitor struct {
	metrics       *PerformanceMetrics
	slowThreshold time.Duration
	startTime     time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(slowThreshold time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: &PerformanceMetrics{
			EndpointStats: make(map[string]Stats),
			SlowQueries:   make([]SlowQuery, 0, 100), // Keep last 100 slow queries
		},
		slowThreshold: slowThreshold,
		startTime:     time.Now(),
	}
}

// PerformanceMiddleware tracks request performance
func (pm *PerformanceMonitor) PerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method
		endpoint := fmt.Sprintf("%s %s", method, path)

		// Skip health check and static files
		if strings.HasPrefix(path, "/static/") || path == "/health" {
			c.Next()
			return
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		status := c.Writer.Status()

		// Update metrics
		pm.updateMetrics(endpoint, duration, status >= 400)

		// Log slow requests
		if duration > pm.slowThreshold {
			log.Printf("SLOW REQUEST: %s %s took %v (status: %d)", 
				method, c.Request.URL.Path, duration, status)
		}

		// Log errors
		if status >= 500 {
			log.Printf("ERROR REQUEST: %s %s returned %d in %v", 
				method, c.Request.URL.Path, status, duration)
		}

		// Add performance headers
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Request-ID", fmt.Sprintf("%d", start.UnixNano()))
	}
}

// updateMetrics updates performance metrics
func (pm *PerformanceMonitor) updateMetrics(endpoint string, duration time.Duration, isError bool) {
	pm.metrics.RequestCount++

	// Update endpoint stats
	stats := pm.metrics.EndpointStats[endpoint]
	stats.Count++
	stats.TotalDuration += duration
	stats.AverageTime = stats.TotalDuration / time.Duration(stats.Count)

	if isError {
		stats.ErrorCount++
	}

	if duration > pm.slowThreshold {
		stats.SlowCount++
	}

	pm.metrics.EndpointStats[endpoint] = stats

	// Update memory stats
	pm.updateMemoryStats()

	// Calculate error rate
	totalErrors := int64(0)
	for _, stat := range pm.metrics.EndpointStats {
		totalErrors += stat.ErrorCount
	}
	pm.metrics.ErrorRate = float64(totalErrors) / float64(pm.metrics.RequestCount) * 100
}

// updateMemoryStats updates memory usage statistics
func (pm *PerformanceMonitor) updateMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.metrics.MemoryUsage = MemoryStats{
		Allocated:    m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		GCRuns:       m.NumGC,
		HeapInUse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
	}
}

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.updateMemoryStats()
	return pm.metrics
}

// GetTopSlowEndpoints returns the slowest endpoints
func (pm *PerformanceMonitor) GetTopSlowEndpoints(limit int) []EndpointSummary {
	endpoints := make([]EndpointSummary, 0, len(pm.metrics.EndpointStats))

	for endpoint, stats := range pm.metrics.EndpointStats {
		endpoints = append(endpoints, EndpointSummary{
			Endpoint:    endpoint,
			AverageTime: stats.AverageTime,
			Count:       stats.Count,
			ErrorRate:   float64(stats.ErrorCount) / float64(stats.Count) * 100,
			SlowRate:    float64(stats.SlowCount) / float64(stats.Count) * 100,
		})
	}

	// Sort by average time (simple bubble sort for small datasets)
	for i := 0; i < len(endpoints)-1; i++ {
		for j := 0; j < len(endpoints)-i-1; j++ {
			if endpoints[j].AverageTime < endpoints[j+1].AverageTime {
				endpoints[j], endpoints[j+1] = endpoints[j+1], endpoints[j]
			}
		}
	}

	if limit > 0 && limit < len(endpoints) {
		endpoints = endpoints[:limit]
	}

	return endpoints
}

// EndpointSummary represents endpoint performance summary
type EndpointSummary struct {
	Endpoint    string        `json:"endpoint"`
	AverageTime time.Duration `json:"average_time"`
	Count       int64         `json:"count"`
	ErrorRate   float64       `json:"error_rate"`
	SlowRate    float64       `json:"slow_rate"`
}

// CompressionMiddleware provides GZIP compression for responses
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for small responses or binary content
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "image/") || 
		   strings.Contains(contentType, "video/") ||
		   strings.Contains(contentType, "application/octet-stream") {
			c.Next()
			return
		}

		// Set compression headers
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		// Create gzip writer
		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		// Replace the writer
		c.Writer = &gzipWriter{Writer: gz, ResponseWriter: c.Writer}
		c.Next()
	}
}

// gzipWriter wraps gin.ResponseWriter with gzip compression
type gzipWriter struct {
	gin.ResponseWriter
	Writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.Writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Writer.Write([]byte(s))
}

// CacheControlMiddleware adds cache headers for static content
func CacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Cache static assets for 1 year
		if strings.HasPrefix(path, "/static/") {
			if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") ||
			   strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") ||
			   strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") ||
			   strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".woff") ||
			   strings.HasSuffix(path, ".woff2") || strings.HasSuffix(path, ".ttf") {
				c.Header("Cache-Control", "public, max-age=31536000") // 1 year
				c.Header("Expires", time.Now().AddDate(1, 0, 0).Format(time.RFC1123))
			}
		} else {
			// No cache for dynamic content
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Don't set HSTS in development
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RequestSizeLimitMiddleware limits request body size
func RequestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(413, gin.H{"error": "Request entity too large"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	clients := make(map[string][]time.Time)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		
		// Clean old requests (older than 1 minute)
		if requests, exists := clients[clientIP]; exists {
			var recent []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < time.Minute {
					recent = append(recent, reqTime)
				}
			}
			clients[clientIP] = recent
		}

		// Check rate limit
		if requests := clients[clientIP]; len(requests) >= requestsPerMinute {
			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		// Add current request
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}

// HealthCheckMiddleware provides health check endpoint
func HealthCheckMiddleware(pm *PerformanceMonitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path != "/health" {
			c.Next()
			return
		}

		metrics := pm.GetMetrics()
		
		// Simple health check
		health := gin.H{
			"status":     "healthy",
			"timestamp":  time.Now(),
			"uptime":     time.Since(pm.startTime).String(),
			"requests":   metrics.RequestCount,
			"error_rate": fmt.Sprintf("%.2f%%", metrics.ErrorRate),
			"memory": gin.H{
				"allocated": formatBytes(metrics.MemoryUsage.Allocated),
				"sys":       formatBytes(metrics.MemoryUsage.Sys),
				"gc_runs":   metrics.MemoryUsage.GCRuns,
			},
		}

		// Add status based on error rate
		if metrics.ErrorRate > 10 {
			health["status"] = "degraded"
		}
		if metrics.ErrorRate > 25 {
			health["status"] = "unhealthy"
		}

		c.JSON(200, health)
		c.Abort()
	}
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

// Global performance monitor instance
var GlobalPerformanceMonitor *PerformanceMonitor

// InitializePerformanceMonitor initializes the global performance monitor
func InitializePerformanceMonitor(slowThreshold time.Duration) {
	GlobalPerformanceMonitor = NewPerformanceMonitor(slowThreshold)
}