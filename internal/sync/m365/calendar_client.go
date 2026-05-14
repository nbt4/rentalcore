package m365

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CalendarEvent struct {
	Subject   string        `json:"subject"`
	Body      EventBody     `json:"body"`
	Start     EventDateTime `json:"start"`
	End       EventDateTime `json:"end"`
	Attendees []Attendee    `json:"attendees,omitempty"`
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

func (c *CalendarClient) CreateEvent(event CalendarEvent) (string, error) {
	resp, err := c.gc.doRequest("POST",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events", c.mailbox),
		event,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create event HTTP %d: %s", resp.StatusCode, body)
	}
	var result createdEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}
	return result.ID, nil
}

func (c *CalendarClient) UpdateEvent(eventID string, event CalendarEvent) error {
	resp, err := c.gc.doRequest("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", c.mailbox, eventID),
		event,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update event HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (c *CalendarClient) DeleteEvent(eventID string) error {
	resp, err := c.gc.doRequest("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/events/%s", c.mailbox, eventID),
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete event HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}
