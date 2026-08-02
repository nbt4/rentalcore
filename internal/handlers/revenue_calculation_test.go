package handlers

import (
	"testing"
	"time"

	"go-barcode-webapp/internal/models"
)

func TestSplitInvoiceRevenue(t *testing.T) {
	tests := []struct {
		name               string
		amount             float64
		taxRate            float64
		pricesIncludeTax   bool
		wantNet, wantGross float64
	}{
		{name: "gross price", amount: 600, taxRate: 19, pricesIncludeTax: true, wantNet: 504.2016807, wantGross: 600},
		{name: "net price", amount: 600, taxRate: 19, pricesIncludeTax: false, wantNet: 600, wantGross: 714},
		{name: "tax free gross job", amount: 600, taxRate: 0, pricesIncludeTax: true, wantNet: 600, wantGross: 600},
		{name: "tax free net job", amount: 600, taxRate: 0, pricesIncludeTax: false, wantNet: 600, wantGross: 600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitInvoiceRevenue(tt.amount, tt.taxRate, tt.pricesIncludeTax)
			assertClose(t, got.Net, tt.wantNet)
			assertClose(t, got.Gross, tt.wantGross)
		})
	}
}

func TestCalculateJobPositionRevenueUsesLivePositions(t *testing.T) {
	start := time.Date(2026, time.May, 28, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	job := models.Job{StartDate: &start, EndDate: &end, MultiplyByDays: false}
	positions := []models.JobPosition{
		{Quantity: 1, UnitPrice: 600, FollowDayFactor: 0.5},
		{Quantity: 2, UnitPrice: 100, FollowDayFactor: 0.5},
	}

	revenue, finalRevenue := calculateJobPositionRevenue(job, positions)
	assertClose(t, revenue, 800)
	assertClose(t, finalRevenue, 800)

	job.MultiplyByDays = true
	revenue, finalRevenue = calculateJobPositionRevenue(job, positions)
	assertClose(t, revenue, 1600)
	assertClose(t, finalRevenue, 1600)
}

func TestCalculateJobPositionRevenueAppliesDiscounts(t *testing.T) {
	job := models.Job{Discount: 10, DiscountType: "percent"}
	positions := []models.JobPosition{{Quantity: 2, UnitPrice: 100, DiscountPercent: 10}}

	revenue, finalRevenue := calculateJobPositionRevenue(job, positions)
	assertClose(t, revenue, 180)
	assertClose(t, finalRevenue, 162)
}

func TestCalculateJobPositionRevenueStoresGrossJobValue(t *testing.T) {
	job := models.Job{PricesIncludeTax: false}
	positions := []models.JobPosition{{Quantity: 1, UnitPrice: 600, TaxRate: 19}}

	revenue, finalRevenue := calculateJobPositionRevenue(job, positions)
	assertClose(t, revenue, 714)
	assertClose(t, finalRevenue, 714)
}
