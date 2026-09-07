package m365

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type EventLocation struct {
	DisplayName string `json:"displayName"`
}

type CalendarEvent struct {
	Subject               string         `json:"subject"`
	Body                  EventBody      `json:"body"`
	Start                 EventDateTime  `json:"start"`
	End                   EventDateTime  `json:"end"`
	Location              *EventLocation `json:"location,omitempty"`
	Attendees             []Attendee     `json:"attendees"`
	IsAllDay              bool           `json:"isAllDay"`
	ShowAs                string         `json:"showAs"`
	ResponseRequested     bool           `json:"responseRequested"`
	AllowNewTimeProposals bool           `json:"allowNewTimeProposals"`
	TransactionID         string         `json:"transactionId,omitempty"`
}

type EventBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type EventDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type Attendee struct {
	EmailAddress EmailAddr `json:"emailAddress"`
	Type         string    `json:"type"`
}

type CreatedEvent struct {
	ID      string `json:"id"`
	ICalUID string `json:"iCalUId"`
}

type eventListResponse struct {
	Value []CreatedEvent `json:"value"`
}

type calendarHTTPError struct {
	operation string
	status    int
	body      string
}

func (e *calendarHTTPError) Error() string {
	return fmt.Sprintf("%s HTTP %d: %s", e.operation, e.status, e.body)
}

func isCalendarNotFound(err error) bool {
	httpErr, ok := err.(*calendarHTTPError)
	return ok && httpErr.status == http.StatusNotFound
}

// CalendarClient bettet GraphClient ein und verwendet seinen Token-Cache.
type CalendarClient struct {
	gc      *GraphClient
	mailbox string
}

func NewCalendarClient(gc *GraphClient, mailbox string) *CalendarClient {
	return &CalendarClient{gc: gc, mailbox: mailbox}
}

func (c *CalendarClient) mailboxHeaders() map[string]string {
	return map[string]string{"x-anchor-mailbox": c.mailbox}
}

func (c *CalendarClient) userHeaders(email string) map[string]string {
	return map[string]string{"x-anchor-mailbox": email}
}

func (c *CalendarClient) DeleteUserEvent(userEmail, eventID string) error {
	resp, err := c.gc.doRequestWithHeaders("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", url.PathEscape(userEmail), url.PathEscape(eventID)),
		nil,
		c.userHeaders(userEmail),
	)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		return &calendarHTTPError{operation: "delete user event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	return nil
}

func (c *CalendarClient) CreateEvent(event CalendarEvent) (CreatedEvent, error) {
	resp, err := c.gc.doRequestWithHeaders("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events", url.PathEscape(c.mailbox)),
		event,
		c.mailboxHeaders(),
	)
	if err != nil {
		return CreatedEvent{}, err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return CreatedEvent{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return CreatedEvent{}, &calendarHTTPError{operation: "create event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	var result CreatedEvent
	if err := json.Unmarshal(bodyBytes, &result); err != nil { // FIXED: use Unmarshal
		return CreatedEvent{}, fmt.Errorf("decode create response: %w", err)
	}
	return result, nil
}

func (c *CalendarClient) UpdateEvent(eventID string, event CalendarEvent) error {
	resp, err := c.gc.doRequestWithHeaders("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", url.PathEscape(c.mailbox), url.PathEscape(eventID)),
		event,
		c.mailboxHeaders(),
	)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return &calendarHTTPError{operation: "update event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	return nil
}

func (c *CalendarClient) GetEvent(eventID string) (CreatedEvent, error) {
	resp, err := c.gc.doRequestWithHeaders("GET",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s?$select=id,iCalUId", url.PathEscape(c.mailbox), url.PathEscape(eventID)),
		nil,
		c.mailboxHeaders(),
	)
	if err != nil {
		return CreatedEvent{}, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return CreatedEvent{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return CreatedEvent{}, &calendarHTTPError{operation: "get event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	var result CreatedEvent
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return CreatedEvent{}, fmt.Errorf("decode event response: %w", err)
	}
	return result, nil
}

func (c *CalendarClient) DeleteEvent(eventID string) error {
	resp, err := c.gc.doRequestWithHeaders("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", url.PathEscape(c.mailbox), url.PathEscape(eventID)),
		nil,
		c.mailboxHeaders(),
	)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		return &calendarHTTPError{operation: "delete event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	return nil
}

func (c *CalendarClient) FindUserEventByICalUID(userEmail, iCalUID string) (string, error) {
	query := url.Values{}
	query.Set("$select", "id")
	query.Set("$filter", fmt.Sprintf("iCalUId eq '%s'", strings.ReplaceAll(iCalUID, "'", "''")))
	query.Set("$top", "1")
	resp, err := c.gc.doRequestWithHeaders("GET",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events?%s", url.PathEscape(userEmail), query.Encode()),
		nil,
		c.userHeaders(userEmail),
	)
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", &calendarHTTPError{operation: "find attendee event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	var result eventListResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("decode attendee events: %w", err)
	}
	if len(result.Value) == 0 {
		return "", &calendarHTTPError{operation: "find attendee event", status: http.StatusNotFound, body: "invitation not processed yet"}
	}
	return result.Value[0].ID, nil
}

func (c *CalendarClient) AcceptUserEvent(userEmail, eventID string) error {
	resp, err := c.gc.doRequestWithHeaders("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s/accept", url.PathEscape(userEmail), url.PathEscape(eventID)),
		map[string]bool{"sendResponse": false},
		c.userHeaders(userEmail),
	)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		return &calendarHTTPError{operation: "accept attendee event", status: resp.StatusCode, body: string(bodyBytes)}
	}
	return nil
}
