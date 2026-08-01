package pricing

import (
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
)

func TestCalculatePositionTotalsInvoiceGross(t *testing.T) {
	job := &models.Job{PricesIncludeTax: false, MultiplyByDays: false}
	positions := []models.JobPosition{
		{Quantity: 2, UnitPrice: 60, TaxRate: 19},
		{Quantity: 2, UnitPrice: 80, TaxRate: 19},
		{Quantity: 1, UnitPrice: 50, TaxRate: 19},
	}

	totals := CalculatePositionTotals(job, positions)

	if totals.Subtotal != 330 || totals.Tax != 62.70 || totals.Gross != 392.70 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
	if totals.GrossBeforeDiscount != 392.70 {
		t.Fatalf("gross before discount = %.2f, want 392.70", totals.GrossBeforeDiscount)
	}
}

func TestCalculatePositionTotalsGrossPricesAndDiscount(t *testing.T) {
	job := &models.Job{PricesIncludeTax: true, Discount: 10, DiscountType: "percent"}
	positions := []models.JobPosition{{Quantity: 1, UnitPrice: 119, TaxRate: 19}}

	totals := CalculatePositionTotals(job, positions)

	if totals.Subtotal != 100 || totals.GlobalDiscount != 10 || totals.Net != 90 || totals.Tax != 17.10 || totals.Gross != 107.10 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestCalculatePositionTotalsFollowDays(t *testing.T) {
	start := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	job := &models.Job{StartDate: &start, EndDate: &end, MultiplyByDays: true}
	positions := []models.JobPosition{{Quantity: 1, UnitPrice: 100, FollowDayFactor: 0.5, TaxRate: 19}}

	totals := CalculatePositionTotals(job, positions)

	if totals.EventDays != 3 || totals.Subtotal != 200 || totals.Gross != 238 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}
