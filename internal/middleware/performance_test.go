package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestPerformanceMonitorConcurrentAccess(t *testing.T) {
	monitor := NewPerformanceMonitor(50 * time.Millisecond)
	const writers = 50
	const updatesPerWriter = 100

	var wg sync.WaitGroup
	for writer := 0; writer < writers; writer++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for update := 0; update < updatesPerWriter; update++ {
				monitor.updateMetrics(
					fmt.Sprintf("GET /jobs/%d", id%5),
					time.Duration(update+1)*time.Millisecond,
					update%10 == 0,
				)
				_ = monitor.GetMetrics()
				_ = monitor.GetTopSlowEndpoints(3)
			}
		}(writer)
	}
	wg.Wait()

	metrics := monitor.GetMetrics()
	if metrics.RequestCount != writers*updatesPerWriter {
		t.Fatalf("request count = %d, want %d", metrics.RequestCount, writers*updatesPerWriter)
	}
	if len(metrics.EndpointStats) != 5 {
		t.Fatalf("endpoint count = %d, want 5", len(metrics.EndpointStats))
	}
}

func TestGetMetricsReturnsIndependentSnapshot(t *testing.T) {
	monitor := NewPerformanceMonitor(time.Second)
	monitor.updateMetrics("GET /jobs", time.Millisecond, false)

	snapshot := monitor.GetMetrics()
	snapshot.EndpointStats["GET /jobs"] = Stats{}

	current := monitor.GetMetrics()
	if current.EndpointStats["GET /jobs"].Count != 1 {
		t.Fatalf("internal metrics were mutated through snapshot: %+v", current.EndpointStats)
	}
}

func TestRateLimitMiddlewareConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(25))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	const requests = 100
	var wg sync.WaitGroup
	statuses := make(chan int, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.RemoteAddr = "192.0.2.10:1234"
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			statuses <- response.Code
		}()
	}
	wg.Wait()
	close(statuses)

	accepted := 0
	limited := 0
	for status := range statuses {
		switch status {
		case http.StatusNoContent:
			accepted++
		case http.StatusTooManyRequests:
			limited++
		default:
			t.Fatalf("unexpected status: %d", status)
		}
	}
	if accepted != 25 || limited != requests-25 {
		t.Fatalf("accepted=%d limited=%d, want 25/%d", accepted, limited, requests-25)
	}
}
