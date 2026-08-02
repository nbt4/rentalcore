package repository

import (
	"testing"

	"go-barcode-webapp/internal/models"
)

func TestAttachJobListRelationsAddsCustomerAndStatus(t *testing.T) {
	company := "Tsunami Events"
	jobs := []models.JobWithDetails{{JobID: 1, CustomerID: 10, StatusID: 20}}
	customers := []models.Customer{{CustomerID: 10, CompanyName: &company}}
	statuses := []models.Status{{StatusID: 20, Status: "Planung"}}

	attachJobListRelations(jobs, customers, statuses)

	if jobs[0].Customer == nil || jobs[0].Customer.GetDisplayName() != company {
		t.Fatalf("customer relation was not attached: %#v", jobs[0].Customer)
	}
	if jobs[0].Status == nil || jobs[0].Status.Status != "Planung" {
		t.Fatalf("status relation was not attached: %#v", jobs[0].Status)
	}
}

func TestAttachJobListRelationsLeavesMissingRelationsEmpty(t *testing.T) {
	jobs := []models.JobWithDetails{{JobID: 1, CustomerID: 10, StatusID: 20}}

	attachJobListRelations(jobs, nil, nil)

	if jobs[0].Customer != nil || jobs[0].Status != nil {
		t.Fatalf("expected missing relations to stay nil: %#v", jobs[0])
	}
}
