package m365

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testCalendarClient(t *testing.T, handler roundTripFunc) *CalendarClient {
	t.Helper()
	graphClient := NewGraphClient("tenant", "client", "secret", "")
	graphClient.token = "test-token"
	graphClient.tokenExpiry = time.Now().Add(time.Hour)
	graphClient.httpClient = &http.Client{Transport: handler}
	return NewCalendarClient(graphClient, "jobs-room@example.com")
}

func TestCreateEventUsesRoomMailboxAsUserCalendar(t *testing.T) {
	client := testCalendarClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", req.Method)
		}
		if req.URL.Path != "/v1.0/users/jobs-room@example.com/events" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		if got := req.Header.Get("x-anchor-mailbox"); got != "jobs-room@example.com" {
			t.Fatalf("x-anchor-mailbox = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if value, exists := payload["responseRequested"]; !exists || value != false {
			t.Fatalf("responseRequested = %#v, exists = %v", value, exists)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"room-event","iCalUId":"meeting-uid"}`)),
			Header:     make(http.Header),
		}, nil
	})

	created, err := client.CreateEvent(CalendarEvent{ResponseRequested: false})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "room-event" || created.ICalUID != "meeting-uid" {
		t.Fatalf("created = %#v", created)
	}
}

func TestAcceptUserEventDoesNotSendResponse(t *testing.T) {
	client := testCalendarClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1.0/users/worker@example.com/events/attendee-event/accept" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		var payload map[string]bool
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["sendResponse"] {
			t.Fatal("sendResponse must be false")
		}
		return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	if err := client.AcceptUserEvent("worker@example.com", "attendee-event"); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAttendeesDeduplicatesAndExcludesRoom(t *testing.T) {
	first := " Worker@example.com "
	duplicate := "worker@EXAMPLE.com"
	room := "jobs-room@example.com"
	employees := []models.JobEmployee{
		{Employee: models.Employee{Email: &first}},
		{Employee: models.Employee{Email: &duplicate}},
		{Employee: models.Employee{Email: &room}},
		{Employee: models.Employee{}},
	}

	attendees := buildAttendees(employees, room)
	if len(attendees) != 1 {
		t.Fatalf("attendees = %#v", attendees)
	}
	if attendees[0].EmailAddress.Address != "Worker@example.com" || attendees[0].Type != "required" {
		t.Fatalf("attendee = %#v", attendees[0])
	}
}

func TestJobTransactionIDIncludesCreationTime(t *testing.T) {
	createdAt := time.Date(2026, time.September, 7, 12, 30, 0, 0, time.UTC)
	first := jobTransactionID(&models.Job{JobID: 42, CreatedAt: &createdAt})
	secondCreatedAt := createdAt.Add(time.Second)
	second := jobTransactionID(&models.Job{JobID: 42, CreatedAt: &secondCreatedAt})
	if first == second {
		t.Fatalf("transaction IDs must differ after a database reset: %q", first)
	}
}
