package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
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
