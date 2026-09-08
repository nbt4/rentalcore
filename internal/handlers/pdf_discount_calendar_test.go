package handlers

import (
	"database/sql"
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
)

func TestExtractionItemDiscountPercent(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int64
		unitPrice float64
		lineTotal float64
		want      float64
	}{
		{name: "full discount", quantity: 1, unitPrice: 250, lineTotal: 0, want: 100},
		{name: "partial discount", quantity: 4, unitPrice: 20, lineTotal: 64, want: 20},
		{name: "no discount", quantity: 2, unitPrice: 10, lineTotal: 20, want: 0},
		{name: "surcharge is not a discount", quantity: 2, unitPrice: 10, lineTotal: 25, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := models.PDFExtractionItem{
				Quantity:  sql.NullInt64{Int64: test.quantity, Valid: true},
				UnitPrice: sql.NullFloat64{Float64: test.unitPrice, Valid: true},
				LineTotal: sql.NullFloat64{Float64: test.lineTotal, Valid: true},
			}
			if got := extractionItemDiscountPercent(item); got != test.want {
				t.Fatalf("extractionItemDiscountPercent() = %.2f, want %.2f", got, test.want)
			}
		})
	}
}

type calendarSyncCapture struct {
	synced chan uint
}

func (s *calendarSyncCapture) SyncJobEvent(jobID uint) {
	s.synced <- jobID
}

func (*calendarSyncCapture) DeleteJobEvent(uint) {}

func (*calendarSyncCapture) SyncEmployeeEvent(uint, uint) {}

func (*calendarSyncCapture) DeleteEmployeeEvent(uint, uint) {}

func TestPDFHandlerSyncJobCalendar(t *testing.T) {
	calendar := &calendarSyncCapture{synced: make(chan uint, 1)}
	handler := &PDFHandler{JobHandler: &JobHandler{calendarSync: calendar}}

	handler.syncJobCalendar(1159)

	select {
	case jobID := <-calendar.synced:
		if jobID != 1159 {
			t.Fatalf("synced job ID = %d, want 1159", jobID)
		}
	case <-time.After(time.Second):
		t.Fatal("calendar sync was not triggered")
	}
}
