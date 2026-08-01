package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	commonjwt "github.com/nbt4/cores-common/pkg/jwt"
)

func TestSetCoresTokenCreatesSharedDomainCookie(t *testing.T) {
	t.Setenv("CORES_JWT_SECRET", "test-secret-with-enough-entropy-for-tests")
	t.Setenv("COOKIE_DOMAIN", ".tsunami-events.de")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "https://rent.tsunami-events.de/api/v1/auth/login", nil)
	user := &models.User{UserID: 20, Username: "mschuck", IsAdmin: true}

	if err := setCoresToken(ctx, user, 3600); err != nil {
		t.Fatalf("setCoresToken returned error: %v", err)
	}

	response := recorder.Result()
	var tokenCookieFound bool
	for _, cookie := range response.Cookies() {
		if cookie.Name != "cores_token" {
			continue
		}
		tokenCookieFound = true
		if strings.TrimPrefix(cookie.Domain, ".") != "tsunami-events.de" {
			t.Fatalf("cookie domain = %q, want .tsunami-events.de", cookie.Domain)
		}
		if !cookie.HttpOnly || !cookie.Secure {
			t.Fatalf("shared cookie must be HttpOnly and Secure: %+v", cookie)
		}
		claims, ok := commonjwt.ValidateToken(cookie.Value)
		if !ok {
			t.Fatal("shared token did not validate")
		}
		if claims.UserID != user.UserID || claims.Username != user.Username || !claims.IsAdmin {
			t.Fatalf("unexpected claims: %+v", claims)
		}
		if !coresTokenBelongsToUser(cookie.Value, user.UserID) {
			t.Fatal("shared token was not recognized for its user")
		}
		if coresTokenBelongsToUser(cookie.Value, user.UserID+1) {
			t.Fatal("shared token was accepted for a different user")
		}
	}
	if !tokenCookieFound {
		t.Fatal("cores_token cookie was not set")
	}
}
