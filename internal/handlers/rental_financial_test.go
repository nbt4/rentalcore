package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go-barcode-webapp/internal/models"
)

func TestRentalFinancialResponsePreservesReceipt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	raw := []byte(`{"job":{"job_id":9007199254740993,"revenue":917.25,"final_revenue":900.25,"discount":17,"discount_type":"amount","multiply_by_days":false,"prices_include_tax":false},"current":{"revenue":800},"draft":{"discount":0},"diff":{"revenue":{"before":800,"after":917.25},"description":{"before":"old","after":"new"}},"expected_context":"unchanged-context","required_confirmation_text":"unchanged-phrase"}`)
	before := string(raw)
	for _, financial := range []bool{false, true, false, true} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		rentalJobFinancialResponse(c, http.StatusCreated, raw, financial)
		if w.Code != http.StatusCreated || string(raw) != before {
			t.Fatal("receipt altered", w.Code)
		}
		if financial {
			if w.Body.String() != before {
				t.Fatal("granted receipt changed", w.Body.String())
			}
			continue
		}
		for _, key := range []string{"revenue", "final_revenue", "discount", "discount_type", "multiply_by_days", "prices_include_tax"} {
			if strings.Contains(w.Body.String(), `"`+key+`"`) {
				t.Fatal("financial field leaked", key, w.Body.String())
			}
		}
		for _, preserved := range []string{"9007199254740993", "unchanged-context", "unchanged-phrase", "description", "cores:rental:financial"} {
			if !strings.Contains(w.Body.String(), preserved) {
				t.Fatal("operational contract changed", w.Body.String())
			}
		}
	}
}

func rentalBoundaryToken(t *testing.T, method jwt.SigningMethod, extra jwt.MapClaims) string {
	t.Helper()
	claims := jwt.MapClaims{"uid": 7, "exp": time.Now().Add(time.Minute).Unix()}
	for k, v := range extra {
		claims[k] = v
	}
	raw, err := jwt.NewWithClaims(method, claims).SignedString([]byte("synthetic-rental-boundary-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRentalDelegationCannotUseNativeSession(t *testing.T) {
	t.Setenv("CORES_JWT_SECRET", "synthetic-rental-boundary-secret")
	gin.SetMode(gin.TestMode)
	h := &AuthHandler{} // Rejections must happen before any session/database lookup.
	for _, algorithm := range []jwt.SigningMethod{jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512} {
		for _, scope := range []any{"cores:rental:update", "", true} {
			for _, credential := range []string{"cookie", "bearer"} {
				for _, path := range []string{"/api/v1/jobs/7", "/api/v1/jobs/7/history", "/api/v1/auth/me", "/api/v1/jobs/7/devices/D1"} {
					r := gin.New()
					r.Use(h.AuthMiddleware())
					r.GET(path, func(*gin.Context) { t.Fatal("delegation escaped to native read") })
					req := httptest.NewRequest("GET", path, nil)
					token := rentalBoundaryToken(t, algorithm, jwt.MapClaims{"mcp_scope": scope, "mcp_financial": true})
					if credential == "cookie" {
						req.AddCookie(&http.Cookie{Name: "cores_token", Value: token})
					} else {
						req.Header.Set("Authorization", "Bearer "+token)
					}
					req.AddCookie(&http.Cookie{Name: "session_id", Value: "must-not-fall-back"})
					req.Header.Set("X-Cores-Origin", "MCP/AI")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					if w.Code != http.StatusForbidden {
						t.Fatal(algorithm.Alg(), scope, credential, path, w.Code)
					}
				}
			}
		}
	}
}

func TestRentalNamedBoundaryAndAssignmentRights(t *testing.T) {
	t.Setenv("CORES_JWT_SECRET", "synthetic-rental-boundary-secret")
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name    string
		claims  jwt.MapClaims
		user    models.User
		origin  string
		allowed bool
	}{
		{"selected", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": true}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "MCP/AI", true},
		{"missing financial", jwt.MapClaims{"mcp_scope": "cores:rental:update"}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "MCP/AI", false},
		{"wrong financial type", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": "true"}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "MCP/AI", false},
		{"wrong action", jwt.MapClaims{"mcp_scope": "cores:rental:create", "mcp_financial": true}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "MCP/AI", false},
		{"revoked admin", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": true}, models.User{UserID: 7, IsActive: true}, "MCP/AI", false},
		{"inactive", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": true}, models.User{UserID: 7, IsAdmin: true}, "MCP/AI", false},
		{"wrong user", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": true}, models.User{UserID: 8, IsAdmin: true, IsActive: true}, "MCP/AI", false},
		{"missing origin", jwt.MapClaims{"mcp_scope": "cores:rental:update", "mcp_financial": true}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "", false},
		{"native UI", jwt.MapClaims{}, models.User{UserID: 7, IsAdmin: true, IsActive: true}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/jobs/:id/devices/:deviceId", func(c *gin.Context) {
				c.Set("user", &tc.user)
				if rentalMCPRequestAllowed(c) && rentalMCPDeviceAssignmentAllowed(c) {
					c.Status(http.StatusNoContent)
				}
			})
			req := httptest.NewRequest("POST", "/api/v1/jobs/7/devices/D1", nil)
			req.AddCookie(&http.Cookie{Name: "cores_token", Value: rentalBoundaryToken(t, jwt.SigningMethodHS256, tc.claims)})
			req.Header.Set("X-Cores-Origin", tc.origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if (w.Code == http.StatusNoContent) != tc.allowed {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	for _, field := range []string{`"revenue":0`, `"discount":0`, `"discount_type":"amount"`, `"multiply_by_days":false`, `"prices_include_tax":false`} {
		r := gin.New()
		r.POST("/jobs/:operation", func(c *gin.Context) { c.Set("user", &models.User{UserID: 7}); (&RentalJobMCP{}).Change(c) })
		req := httptest.NewRequest("POST", "/jobs/create", strings.NewReader(`{`+field+`}`))
		req.AddCookie(&http.Cookie{Name: "cores_token", Value: rentalBoundaryToken(t, jwt.SigningMethodHS256, jwt.MapClaims{"mcp_scope": "cores:rental:create"})})
		req.Header.Set("X-Cores-Origin", "MCP/AI")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatal("financial input reached DB", field, w.Code)
		}
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
}
