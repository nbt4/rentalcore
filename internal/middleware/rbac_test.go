package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
)

func TestRequireAdminAcceptsAdminFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	rbac := NewRBACMiddleware(nil)
	router.GET("/admin", func(c *gin.Context) {
		c.Set("user", &models.User{Username: "mschuck", IsAdmin: true})
		c.Next()
	}, rbac.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
