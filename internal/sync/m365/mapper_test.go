package m365_test

import (
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
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
