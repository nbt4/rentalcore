package m365

import "testing"

func TestWithCalendarRecipient(t *testing.T) {
	event := withCalendarRecipient(CalendarEvent{}, "events@tsunami-events.de")
	if len(event.Attendees) != 1 || event.Attendees[0].EmailAddress.Address != "events@tsunami-events.de" {
		t.Fatalf("unexpected attendees: %+v", event.Attendees)
	}

	event = withCalendarRecipient(event, "EVENTS@tsunami-events.de")
	if len(event.Attendees) != 1 {
		t.Fatalf("recipient was added twice: %+v", event.Attendees)
	}
}
