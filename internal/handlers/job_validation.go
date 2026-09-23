package handlers

import (
	"fmt"
	"strings"
	"time"

	"go-barcode-webapp/internal/jobstatus"
	"go-barcode-webapp/internal/models"
)

func parseJobDate(data map[string]interface{}, keys ...string) (*time.Time, bool, error) {
	for _, key := range keys {
		raw, exists := data[key]
		if !exists {
			continue
		}
		if raw == nil || raw == "" {
			return nil, true, nil
		}
		value, ok := raw.(string)
		if !ok {
			return nil, true, fmt.Errorf("Ungültiges Datum")
		}
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, true, fmt.Errorf("Ungültiges Datum: %s", value)
		}
		return &parsed, true, nil
	}
	return nil, false, nil
}

func validateJobWrite(job *models.Job, previousStatus uint) error {
	if job.CustomerID == 0 {
		return fmt.Errorf("Bitte einen Kunden auswählen")
	}
	if job.Description == nil || strings.TrimSpace(*job.Description) == "" {
		return fmt.Errorf("Bitte einen Jobtitel eingeben")
	}
	title := strings.TrimSpace(*job.Description)
	job.Description = &title
	if err := jobstatus.ValidateTransition(previousStatus, job.StatusID); err != nil {
		return fmt.Errorf("Dieser Statuswechsel ist nicht zulässig")
	}
	if (job.StartDate == nil) != (job.EndDate == nil) {
		return fmt.Errorf("Bitte Start- und Enddatum gemeinsam angeben")
	}
	if job.StartDate != nil && job.EndDate.Before(*job.StartDate) {
		return fmt.Errorf("Das Enddatum darf nicht vor dem Startdatum liegen")
	}
	if job.StatusID == jobstatus.ConfirmedID || job.StatusID == jobstatus.CompletedID {
		if job.StartDate == nil {
			return fmt.Errorf("Für einen bestätigten Job ist ein Zeitraum erforderlich")
		}
	}
	if job.Discount < 0 || job.Revenue < 0 {
		return fmt.Errorf("Umsatz und Rabatt dürfen nicht negativ sein")
	}
	if job.DiscountType == "" {
		job.DiscountType = "amount"
	}
	if job.DiscountType != "amount" && job.DiscountType != "percent" {
		return fmt.Errorf("Ungültige Rabattart")
	}
	if job.DiscountType == "percent" && job.Discount > 100 {
		return fmt.Errorf("Prozentrabatt darf höchstens 100 betragen")
	}
	return nil
}
