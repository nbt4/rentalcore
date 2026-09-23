package handlers

import (
	"testing"
	"time"

	"go-barcode-webapp/internal/jobstatus"
	"go-barcode-webapp/internal/models"
)

func TestValidateJobWrite(t *testing.T) {
	start := time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 2)
	title := "  Stadtfest  "
	tests := []struct {
		name      string
		job       models.Job
		previous  uint
		wantError bool
	}{
		{"valid draft", models.Job{CustomerID: 1, StatusID: jobstatus.PlanningID, Description: &title}, jobstatus.PlanningID, false},
		{"confirmed with dates", models.Job{CustomerID: 1, StatusID: jobstatus.ConfirmedID, Description: &title, StartDate: &start, EndDate: &end}, jobstatus.PlanningID, false},
		{"confirmed without dates", models.Job{CustomerID: 1, StatusID: jobstatus.ConfirmedID, Description: &title}, jobstatus.PlanningID, true},
		{"end before start", models.Job{CustomerID: 1, StatusID: jobstatus.PlanningID, Description: &title, StartDate: &end, EndDate: &start}, jobstatus.PlanningID, true},
		{"missing customer", models.Job{StatusID: jobstatus.PlanningID, Description: &title}, jobstatus.PlanningID, true},
		{"invalid transition", models.Job{CustomerID: 1, StatusID: jobstatus.CompletedID, Description: &title, StartDate: &start, EndDate: &end}, jobstatus.PlanningID, true},
		{"negative revenue", models.Job{CustomerID: 1, StatusID: jobstatus.PlanningID, Description: &title, Revenue: -1}, jobstatus.PlanningID, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJobWrite(&tt.job, tt.previous)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateJobWrite() error = %v, wantError %v", err, tt.wantError)
			}
			if err == nil && *tt.job.Description != "Stadtfest" {
				t.Fatalf("title was not trimmed: %q", *tt.job.Description)
			}
		})
	}
}

func TestParseJobDate(t *testing.T) {
	for _, tc := range []struct {
		value       interface{}
		wantError   bool
		wantPresent bool
	}{
		{"2026-10-10", false, true},
		{"2026-02-30", true, true},
		{nil, false, true},
		{12, true, true},
	} {
		date, present, err := parseJobDate(map[string]interface{}{"startDate": tc.value}, "startDate")
		if present != tc.wantPresent || (err != nil) != tc.wantError {
			t.Fatalf("parseJobDate(%v) = (%v, %v, %v)", tc.value, date, present, err)
		}
	}
}
