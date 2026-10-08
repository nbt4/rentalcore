package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newAuthHandlerTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm database: %v", err)
	}
	return db, mock
}

func expectLoginAPISetup(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 AND is_active = \$2 ORDER BY "users"\."userid" LIMIT \$3`).
		WithArgs("test-user", true, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"userid", "username", "email", "password_hash", "first_name", "last_name", "is_active",
		}).AddRow(2, "test-user", "test@example.invalid", string(passwordHash), "Test", "User", true))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "sessions"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}

func newLoginAPIContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{
		"username": "test-user",
		"password": "test-password"
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	return context, recorder
}

func TestLoginAPIQueriesLegacyUserRolesColumn(t *testing.T) {
	db, mock := newAuthHandlerTestDB(t)
	expectLoginAPISetup(t, mock)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE userid = $1`)).
		WithArgs(uint(2)).
		WillReturnRows(sqlmock.NewRows([]string{"userid", "roleid", "is_active"}).AddRow(2, 3, true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE "roles"."roleid" = $1`)).
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"roleid", "name", "display_name", "permissions", "is_active"}).
			AddRow(3, "manager", "Manager", []byte(`[]`), true))

	context, recorder := newLoginAPIContext(t)
	NewAuthHandler(db, &config.Config{}).LoginAPI(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("LoginAPI status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		User struct {
			Roles []struct {
				ID   uint   `json:"id"`
				Name string `json:"name"`
			} `json:"Roles"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode LoginAPI response: %v", err)
	}
	if len(response.User.Roles) != 1 || response.User.Roles[0].ID != 3 || response.User.Roles[0].Name != "manager" {
		t.Fatalf("LoginAPI roles = %#v, want manager role", response.User.Roles)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestLoginAPIReturnsServerErrorWhenRolesCannotBeLoaded(t *testing.T) {
	db, mock := newAuthHandlerTestDB(t)
	expectLoginAPISetup(t, mock)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE userid = $1`)).
		WithArgs(uint(2)).
		WillReturnError(errors.New("role lookup failed"))

	context, recorder := newLoginAPIContext(t)
	NewAuthHandler(db, &config.Config{}).LoginAPI(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("LoginAPI status = %d, want %d; body = %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestMeAPIQueriesLegacyUserRolesColumn(t *testing.T) {
	db, mock := newAuthHandlerTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE userid = $1`)).
		WithArgs(uint(2)).
		WillReturnRows(sqlmock.NewRows([]string{"userid", "roleid", "is_active"}).AddRow(2, 3, true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE "roles"."roleid" = $1`)).
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"roleid", "name", "display_name", "permissions", "is_active"}).
			AddRow(3, "manager", "Manager", []byte(`[]`), true))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("user", &models.User{UserID: 2, Username: "test-user", IsActive: true})

	NewAuthHandler(db, &config.Config{}).MeAPI(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("MeAPI status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Roles []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"Roles"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode MeAPI response: %v", err)
	}
	if len(response.Roles) != 1 || response.Roles[0].ID != 3 || response.Roles[0].Name != "manager" {
		t.Fatalf("MeAPI roles = %#v, want manager role", response.Roles)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestMeAPIReturnsServerErrorWhenRolesCannotBeLoaded(t *testing.T) {
	db, mock := newAuthHandlerTestDB(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_roles" WHERE userid = $1`)).
		WithArgs(uint(2)).
		WillReturnError(errors.New("role lookup failed"))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("user", &models.User{UserID: 2, Username: "test-user", IsActive: true})

	NewAuthHandler(db, &config.Config{}).MeAPI(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("MeAPI status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
