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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create user event HTTP %d: %s", resp.StatusCode, body)
	}
	var result createdEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update user event HTTP %d: %s", resp.StatusCode, body)
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete user event HTTP %d: %s", resp.StatusCode, body)
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
	resp, err := c.gc.doRequestWithHeaders("PATCH",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/events/%s", c.mailbox, eventID),
		event,
		c.groupHeaders(),
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
	resp, err := c.gc.doRequestWithHeaders("DELETE",
		fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/events/%s", c.mailbox, eventID),
		nil,
		c.groupHeaders(),
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
