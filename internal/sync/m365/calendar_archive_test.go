package m365

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go-barcode-webapp/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCleanupArchivedJobEventsDeletesSharedEvent(t *testing.T) {
	matcher := sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		if strings.Contains(actual, `"jobs"."deleted_at" IS NULL`) {
			return fmt.Errorf("archived job was excluded: %s", actual)
		}
		if !strings.Contains(actual, expected) {
			return fmt.Errorf("query %q does not contain %q", actual, expected)
		}
		return nil
	})
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}

	jobID := uint(42)
	mock.ExpectQuery("deleted_at IS NOT NULL").WillReturnRows(sqlmock.NewRows([]string{"jobid"}).AddRow(jobID))
	mock.ExpectQuery(`FROM "jobs"`).WillReturnRows(sqlmock.NewRows([]string{"jobid", "m365_event_id", "deleted_at"}).
		AddRow(jobID, "room-event", time.Now()))
	mock.ExpectQuery(`FROM "job_employees"`).WillReturnRows(sqlmock.NewRows([]string{"job_id", "employee_id"}))
	mock.ExpectExec(`UPDATE "jobs"`).WithArgs(nil, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	deletes := 0
	client := testCalendarClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete || req.URL.Path != "/v1.0/users/jobs-room@example.com/events/room-event" {
			t.Fatalf("unexpected Graph request: %s %s", req.Method, req.URL.Path)
		}
		deletes++
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	svc := NewCalendarSyncService(client, repository.NewJobRepository(&repository.Database{DB: db}), nil,
		repository.NewJobEmployeeRepository(db), "")
	svc.CleanupArchivedJobEvents()
	if deletes != 1 {
		t.Fatalf("Graph DELETE count = %d, want 1", deletes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
