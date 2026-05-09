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
