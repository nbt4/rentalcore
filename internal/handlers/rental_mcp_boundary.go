package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Inspect credentials before session fallback: a delegated token must never
// become an unrestricted browser session, even when accompanied by a UI cookie.
func rentalMCPDelegation(c *gin.Context) (jwt.MapClaims, bool, error) {
	tokens := []string{}
	if cookie, err := c.Cookie("cores_token"); err == nil && cookie != "" {
		tokens = append(tokens, cookie)
	}
	if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
		tokens = append(tokens, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	}
	var delegation jwt.MapClaims
	for _, raw := range tokens {
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
			return []byte(os.Getenv("CORES_JWT_SECRET")), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if _, delegated := claims["mcp_scope"]; !delegated {
			continue
		}
		scope, ok := claims["mcp_scope"].(string)
		expires, expiresErr := claims.GetExpirationTime()
		if err != nil || token == nil || !token.Valid || os.Getenv("CORES_JWT_SECRET") == "" || !ok || scope == "" || expiresErr != nil || expires == nil {
			return nil, true, fmt.Errorf("Valid signed expiring MCP action delegation required")
		}
		// Multiple delegated credentials are ambiguous and cannot fall back to UI.
		if delegation != nil {
			return nil, true, fmt.Errorf("One MCP action delegation required")
		}
		delegation = claims
	}
	return delegation, delegation != nil, nil
}

func rentalMCPRequestAllowed(c *gin.Context) bool {
	_, delegated, err := rentalMCPDelegation(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return false
	}
	if !delegated {
		return true
	}
	path := c.Request.URL.Path
	allowed := c.Request.Method == http.MethodPost && (strings.HasPrefix(path, "/api/v1/mcp/") || c.FullPath() == "/api/v1/jobs/:id/devices/:deviceId")
	if !allowed || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "MCP delegation is limited to named Rental workflows"})
		return false
	}
	return true
}

func rentalMCPDeviceAssignmentAllowed(c *gin.Context) bool {
	claims, delegated, err := rentalMCPDelegation(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return false
	}
	if !delegated {
		return true
	}
	user, ok := GetCurrentUser(c)
	financial, _ := claims["mcp_financial"].(bool)
	uid, uidOK := claims["uid"].(float64)
	if !ok || user == nil || !user.IsActive || !user.IsAdmin || !uidOK || uid != float64(user.UserID) || uid == 0 || claims["mcp_scope"] != "cores:rental:update" || !financial || c.GetHeader("X-Cores-Origin") != "MCP/AI" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Current Rental administrator, selected update action and separate Rental financial access required"})
		return false
	}
	return true
}
