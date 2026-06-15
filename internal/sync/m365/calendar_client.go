package m365

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type EventLocation struct {
	DisplayName string `json:"displayName"`
}

type CalendarEvent struct {
	Subject   string          `json:"subject"`
	Body      EventBody       `json:"body"`
	Start     EventDateTime   `json:"start"`
	End       EventDateTime   `json:"end"`
	Location  *EventLocation  `json:"location,omitempty"`
	Attendees []Attendee      `json:"attendees,omitempty"`
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

type createdEventResponse struct {
	ID string `json:"id"`
}

// CalendarClient bettet GraphClient ein und verwendet seinen Token-Cache.
type CalendarClient struct {
	gc      *GraphClient
	mailbox string
}

func NewCalendarClient(gc *GraphClient, mailbox string) *CalendarClient {
	return &CalendarClient{gc: gc, mailbox: mailbox}
}

func (c *CalendarClient) groupHeaders() map[string]string {
	return map[string]string{
		"x-anchor-mailbox": "GroupMailbox:" + c.mailbox,
	}
}

func (c *CalendarClient) userHeaders(email string) map[string]string {
	return map[string]string{"x-anchor-mailbox": email}
}

func (c *CalendarClient) CreateUserEvent(userEmail string, event CalendarEvent) (string, error) {
	resp, err := c.gc.doRequestWithHeaders("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events", userEmail),
		event,
		c.userHeaders(userEmail),
	)
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create user event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var result createdEventResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil { // FIXED: use Unmarshal
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.ID, nil
}

func (c *CalendarClient) UpdateUserEvent(userEmail, eventID string, event CalendarEvent) error {
	resp, err := c.gc.doRequestWithHeaders("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", userEmail, eventID),
		event,
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update user event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *CalendarClient) DeleteUserEvent(userEmail, eventID string) error {
	resp, err := c.gc.doRequestWithHeaders("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", userEmail, eventID),
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
		return fmt.Errorf("delete user event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *CalendarClient) CreateEvent(event CalendarEvent) (string, error) {
	resp, err := c.gc.doRequestWithHeaders("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/events", c.mailbox),
		event,
		c.groupHeaders(),
	)
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body) // FIXED: read body once, reuse
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var result createdEventResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil { // FIXED: use Unmarshal
		return "", fmt.Errorf("decode create response: %w", err)
	}
	return result.ID, nil
}

func (c *CalendarClient) UpdateEvent(eventID string, event CalendarEvent) error {
	resp, err := c.gc.doRequestWithHeaders("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/events/%s", c.mailbox, eventID),
		event,
		c.groupHeaders(),
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
		return fmt.Errorf("update event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *CalendarClient) DeleteEvent(eventID string) error {
	resp, err := c.gc.doRequestWithHeaders("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/events/%s", c.mailbox, eventID),
		nil,
		c.groupHeaders(),
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
		return fmt.Errorf("delete event HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
