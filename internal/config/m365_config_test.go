package config

import "testing"

func TestM365ServicesCanBeConfiguredIndependently(t *testing.T) {
	tests := []struct {
		name               string
		config             M365Config
		configured         bool
		contactsConfigured bool
		calendarConfigured bool
	}{
		{name: "missing credentials", config: M365Config{MailboxID: "contacts@example.com", CalendarMailbox: "events@example.com"}},
		{name: "contacts only", config: M365Config{TenantID: "tenant", ClientID: "client", ClientSecret: "secret", MailboxID: "contacts@example.com"}, configured: true, contactsConfigured: true},
		{name: "calendar only", config: M365Config{TenantID: "tenant", ClientID: "client", ClientSecret: "secret", CalendarMailbox: "events@example.com"}, configured: true, calendarConfigured: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.IsConfigured(); got != test.configured {
				t.Fatalf("IsConfigured() = %v, want %v", got, test.configured)
			}
			if got := test.config.ContactsConfigured(); got != test.contactsConfigured {
				t.Fatalf("ContactsConfigured() = %v, want %v", got, test.contactsConfigured)
			}
			if got := test.config.CalendarConfigured(); got != test.calendarConfigured {
				t.Fatalf("CalendarConfigured() = %v, want %v", got, test.calendarConfigured)
			}
		})
	}
}
