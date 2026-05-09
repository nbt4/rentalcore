package m365_test

import (
	"os"
	"testing"
	"time"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/models"
	m365 "go-barcode-webapp/internal/sync/m365"
)

func TestCustomerHasM365Fields(t *testing.T) {
	c := models.Customer{}
	c.M365ID = testStrPtr("test-id")
	c.IsArchived = true

	now := time.Now()
	c.M365UpdatedAt = &now
	c.UpdatedAt = now

	if c.M365ID == nil || *c.M365ID != "test-id" {
		t.Error("M365ID field missing or wrong")
	}
	if !c.IsArchived {
		t.Error("IsArchived field missing")
	}
}

func testStrPtr(s string) *string { return &s }

func TestNewGraphClientRequiresAllFields(t *testing.T) {
	client := m365.NewGraphClient("tid", "cid", "csec", "mbx")
	if client == nil {
		t.Fatal("NewGraphClient returned nil")
	}
}

func TestExtractDeltaToken(t *testing.T) {
	link := "https://graph.microsoft.com/v1.0/users/mb/contacts/delta?$deltaToken=TOKEN123"
	got := m365.ExtractDeltaToken(link)
	if got != "TOKEN123" {
		t.Errorf("got %q, want %q", got, "TOKEN123")
	}
}

func TestCustomerToContact(t *testing.T) {
	company := "Acme GmbH"
	first := "Max"
	last := "Mustermann"
	email := "max@acme.de"
	phone := "+49 123 456"
	street := "Hauptstraße"
	house := "42"
	zip := "70173"
	city := "Stuttgart"
	country := "Deutschland"
	notes := "VIP-Kunde"

	c := models.Customer{
		CompanyName: &company,
		FirstName:   &first,
		LastName:    &last,
		Email:       &email,
		PhoneNumber: &phone,
		Street:      &street,
		HouseNumber: &house,
		ZIP:         &zip,
		City:        &city,
		Country:     &country,
		Notes:       &notes,
	}

	contact := m365.CustomerToContact(&c)

	if contact.CompanyName != "Acme GmbH" {
		t.Errorf("CompanyName: got %q", contact.CompanyName)
	}
	if contact.GivenName != "Max" {
		t.Errorf("GivenName: got %q", contact.GivenName)
	}
	if len(contact.EmailAddresses) == 0 || contact.EmailAddresses[0].Address != "max@acme.de" {
		t.Error("EmailAddresses not mapped correctly")
	}
	if contact.BusinessAddress.Street != "Hauptstraße 42" {
		t.Errorf("Street+HouseNumber: got %q", contact.BusinessAddress.Street)
	}
	if contact.BusinessAddress.PostalCode != "70173" {
		t.Errorf("PostalCode: got %q", contact.BusinessAddress.PostalCode)
	}
	if contact.PersonalNotes != "VIP-Kunde" {
		t.Errorf("PersonalNotes: got %q", contact.PersonalNotes)
	}
}

func TestContactToCustomer(t *testing.T) {
	contact := m365.M365Contact{
		ID:             "abc-123",
		GivenName:      "Anna",
		Surname:        "Schmidt",
		CompanyName:    "Schmidt AG",
		EmailAddresses: []m365.EmailAddr{{Address: "anna@schmidt.de"}},
		BusinessPhones: []string{"+49 711 999"},
		BusinessAddress: m365.Address{
			Street:          "Königstraße 10",
			PostalCode:      "70173",
			City:            "Stuttgart",
			CountryOrRegion: "Deutschland",
		},
		PersonalNotes:        "Notiz",
		LastModifiedDateTime: "2026-05-09T10:00:00Z",
	}

	c := m365.ContactToCustomer(contact)

	if c.FirstName == nil || *c.FirstName != "Anna" {
		t.Error("FirstName not mapped")
	}
	if c.Street == nil || *c.Street != "Königstraße" {
		t.Errorf("Street: got %v", c.Street)
	}
	if c.HouseNumber == nil || *c.HouseNumber != "10" {
		t.Errorf("HouseNumber: got %v", c.HouseNumber)
	}
	if c.M365UpdatedAt == nil {
		t.Error("M365UpdatedAt not set")
	}
}

func TestSplitStreetNumber(t *testing.T) {
	cases := []struct {
		input      string
		wantStreet string
		wantNumber string
	}{
		{"Hauptstraße 42", "Hauptstraße", "42"},
		{"Königstraße 10", "Königstraße", "10"},
		{"Am Marktplatz", "Am Marktplatz", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		street, num := m365.SplitStreetAndNumber(tc.input)
		if street != tc.wantStreet || num != tc.wantNumber {
			t.Errorf("SplitStreetAndNumber(%q): got (%q, %q), want (%q, %q)",
				tc.input, street, num, tc.wantStreet, tc.wantNumber)
		}
	}
}

func TestM365ConfigLoadsFromEnv(t *testing.T) {
	os.Setenv("M365_TENANT_ID", "tenant-123")
	os.Setenv("M365_CLIENT_ID", "client-456")
	os.Setenv("M365_CLIENT_SECRET", "secret-789")
	os.Setenv("M365_SHARED_MAILBOX_ID", "mailbox@test.de")
	os.Setenv("M365_SYNC_INTERVAL", "10m")
	defer func() {
		os.Unsetenv("M365_TENANT_ID")
		os.Unsetenv("M365_CLIENT_ID")
		os.Unsetenv("M365_CLIENT_SECRET")
		os.Unsetenv("M365_SHARED_MAILBOX_ID")
		os.Unsetenv("M365_SYNC_INTERVAL")
	}()

	cfg := config.M365Config{}
	cfg.LoadFromEnv()

	if cfg.TenantID != "tenant-123" {
		t.Errorf("TenantID: got %q, want %q", cfg.TenantID, "tenant-123")
	}
	if cfg.SyncInterval != "10m" {
		t.Errorf("SyncInterval: got %q, want %q", cfg.SyncInterval, "10m")
	}
	if !cfg.IsConfigured() {
		t.Error("IsConfigured() should return true when all fields set")
	}
}
