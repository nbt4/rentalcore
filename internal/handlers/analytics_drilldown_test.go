package handlers

import (
	"math"
	"testing"
	"time"

	"go-barcode-webapp/internal/jobstatus"
)

func TestBuildRevenueDrilldownReconcilesRevenueAndRentalMargin(t *testing.T) {
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	productID, rentalID, serviceID := uint(10), uint(20), uint(30)

	result := buildRevenueDrilldown(
		"realized", "all",
		nil,
		nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 1000, StartDate: &start, EndDate: &end}},
		[]revenueDrilldownPosition{
			{PositionID: 1, JobID: 1, PositionType: "product", ProductID: &productID, ItemName: "Lautsprecher", Quantity: 2, UnitPrice: 200},
			{PositionID: 2, JobID: 1, PositionType: "rental", RentalEquipmentID: &rentalID, ItemName: "Funkstrecke", Quantity: 1, UnitPrice: 300, SupplierUnitCost: 50},
			{PositionID: 3, JobID: 1, PositionType: "service", ServiceItemID: &serviceID, ItemName: "Techniker", Quantity: 1, UnitPrice: 300},
		},
		[]revenueDrilldownDevice{{PositionID: 1, DeviceID: "DEV-1", SerialNumber: "SN-1"}},
		[]revenueDrilldownRentalCost{{JobID: 1, EquipmentID: rentalID, ItemName: "Funkstrecke", TotalCost: 120}},
	)

	assertClose(t, result.TotalRevenue, 1000)
	assertClose(t, result.AttributedRevenue, 1000)
	assertClose(t, result.RentalRevenue, 300)
	assertClose(t, result.RentalCost, 120)
	assertClose(t, result.RentalMargin, 180)

	productCategory := result.Categories[0]
	if len(productCategory.Children) != 1 || len(productCategory.Children[0].Children) != 2 {
		t.Fatalf("expected product and assigned/unassigned device levels, got %#v", productCategory.Children)
	}
	assertClose(t, productCategory.Children[0].Children[0].Revenue+productCategory.Children[0].Children[1].Revenue, 400)
}

func TestBuildRevenueDrilldownKeepsUnattributedRevenueVisible(t *testing.T) {
	result := buildRevenueDrilldown(
		"realized", "all",
		nil,
		nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 250}, {JobID: 2, Revenue: 50}},
		[]revenueDrilldownPosition{
			{PositionID: 1, JobID: 1, PositionType: "product", ItemName: "Produkt", Quantity: 1, UnitPrice: 100},
			{PositionID: 2, JobID: 1, PositionType: "service", ItemName: "Service", Quantity: 1, UnitPrice: 100},
		},
		nil,
		nil,
	)

	assertClose(t, result.TotalRevenue, 300)
	assertClose(t, result.AttributedRevenue, 200)
	assertClose(t, result.UnattributedRevenue, 100)
	assertClose(t, result.Categories[0].Revenue, 100)
	assertClose(t, result.Categories[2].Revenue, 100)
	assertClose(t, result.Categories[4].Revenue, 100)
}

func TestBuildRevenueDrilldownKeepsInvoicePositionPriceAndSplitsTax(t *testing.T) {
	rentalID := uint(4)
	result := buildRevenueDrilldown(
		"realized", "all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1148, Revenue: 531.41, PricesIncludeTax: true}},
		[]revenueDrilldownPosition{{
			PositionID: 29, JobID: 1148, PositionType: "rental", RentalEquipmentID: &rentalID,
			ItemName: "Mikrofone 8x Funkmikrofon 2x Headset", Quantity: 1, UnitPrice: 600, TaxRate: 19,
		}},
		nil, nil,
	)

	assertClose(t, result.TotalGrossRevenue, 600)
	assertClose(t, result.TotalNetRevenue, 504.20)
	assertClose(t, result.AttributedGrossRevenue, 600)
	assertClose(t, result.AttributedNetRevenue, 504.20)
	microphones := result.Categories[1].Children[0]
	assertClose(t, microphones.GrossRevenue, 600)
	assertClose(t, microphones.NetRevenue, 504.20)
	assertClose(t, microphones.TaxAmount, 95.80)
}

func TestBuildRevenueDrilldownConvertsNetAndTaxFreePositions(t *testing.T) {
	result := buildRevenueDrilldown(
		"realized", "all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 700, PricesIncludeTax: false}},
		[]revenueDrilldownPosition{
			{PositionID: 1, JobID: 1, PositionType: "service", ItemName: "Technik", Quantity: 1, UnitPrice: 600, TaxRate: 19},
			{PositionID: 2, JobID: 1, PositionType: "service", ItemName: "Steuerfrei", Quantity: 1, UnitPrice: 100, TaxRate: 0},
		},
		nil, nil,
	)

	assertClose(t, result.TotalNetRevenue, 700)
	assertClose(t, result.TotalGrossRevenue, 814)
	assertClose(t, result.TotalTaxAmount, 114)
	assertClose(t, result.Categories[2].NetRevenue, 700)
}

func TestBuildRevenueDrilldownRentalFallbackFollowsJobDaySetting(t *testing.T) {
	start := time.Date(2026, time.May, 28, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	rentalID := uint(4)
	position := revenueDrilldownPosition{
		PositionID:        1,
		JobID:             1,
		PositionType:      "rental",
		RentalEquipmentID: &rentalID,
		ItemName:          "Mikrofone 8x Funkmikrofon 2x Headset",
		Quantity:          1,
		UnitPrice:         600,
		SupplierUnitCost:  474.74,
	}

	flat := buildRevenueDrilldown(
		"realized", "all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 600, StartDate: &start, EndDate: &end, MultiplyByDays: false}},
		[]revenueDrilldownPosition{position}, nil, nil,
	)
	assertClose(t, flat.RentalCost, 474.74)

	perDay := buildRevenueDrilldown(
		"realized", "all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 600, StartDate: &start, EndDate: &end, MultiplyByDays: true}},
		[]revenueDrilldownPosition{position}, nil, nil,
	)
	assertClose(t, perDay.RentalCost, 1424.22)
}

func TestRevenueDrilldownPeriodRejectsUnknownValue(t *testing.T) {
	if _, _, err := revenueDrilldownPeriod("quarter", "realized", time.Now()); err == nil {
		t.Fatal("expected unsupported period to fail")
	}
}

func TestRevenueDrilldownScopeSeparatesRealizedAndPipeline(t *testing.T) {
	realized, err := revenueDrilldownScope("realized")
	if err != nil || len(realized) != 1 || realized[0] != jobstatus.CompletedID {
		t.Fatalf("realized status selection = %v, %v", realized, err)
	}
	pipeline, err := revenueDrilldownScope("pipeline")
	if err != nil || len(pipeline) != 2 || pipeline[0] != jobstatus.PlanningID || pipeline[1] != jobstatus.ConfirmedID {
		t.Fatalf("pipeline status selection = %v, %v", pipeline, err)
	}
	if _, err := revenueDrilldownScope("all"); err == nil {
		t.Fatal("an unlabelled mixture of realized and pipeline revenue must be rejected")
	}

	now := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.UTC)
	today := time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC)
	pastStart, pastEnd, err := revenueDrilldownPeriod("30days", "realized", now)
	if err != nil || !pastStart.Equal(today.AddDate(0, 0, -30)) || !pastEnd.Equal(today) {
		t.Fatalf("realized period = %v..%v, %v", pastStart, pastEnd, err)
	}
	futureStart, futureEnd, err := revenueDrilldownPeriod("30days", "pipeline", now)
	if err != nil || !futureStart.Equal(today) || !futureEnd.Equal(today.AddDate(0, 0, 30)) {
		t.Fatalf("pipeline period = %v..%v, %v", futureStart, futureEnd, err)
	}
}

func TestRevenueDrilldownServiceListsContributingJobs(t *testing.T) {
	serviceID := uint(42)
	jobs := []revenueDrilldownJob{
		{JobID: 1, JobCode: "JOB000001", JobTitle: "Gala", Revenue: 100},
		{JobID: 2, JobCode: "JOB000002", JobTitle: "Konzert", Revenue: 100},
		{JobID: 3, JobCode: "JOB000003", JobTitle: "Konferenz", Revenue: 100},
	}
	positions := []revenueDrilldownPosition{
		{PositionID: 11, JobID: 1, PositionType: "service", ServiceItemID: &serviceID, ItemName: "Audio Engineer", Quantity: 1, UnitPrice: 100},
		{PositionID: 12, JobID: 2, PositionType: "service", ServiceItemID: &serviceID, ItemName: "Audio Engineer", Quantity: 1, UnitPrice: 100},
		{PositionID: 13, JobID: 3, PositionType: "service", ServiceItemID: &serviceID, ItemName: "Audio Engineer", Quantity: 1, UnitPrice: 100},
	}
	result := buildRevenueDrilldown("realized", "all", nil, nil, jobs, positions, nil, nil)
	node := result.Categories[2].Children[0]
	if node.Bookings != 3 || len(node.Jobs) != 3 {
		t.Fatalf("Audio Engineer must list all three jobs: %+v", node)
	}
	if node.Jobs[0].JobCode != "JOB000003" || node.Jobs[0].JobTitle != "Konferenz" || node.Jobs[0].PositionLabel != "Audio Engineer" {
		t.Fatalf("job details are missing from drilldown: %+v", node.Jobs[0])
	}
}

func TestRevenueDrilldownMonthlyRevenueUsesCompletionDate(t *testing.T) {
	start := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	jobs := []revenueDrilldownJob{{JobID: 1, Revenue: 100, StartDate: &start, EndDate: &end}}
	realized := buildRevenueDrilldown("realized", "all", nil, nil, jobs, nil, nil, nil)
	if len(realized.MonthlyRevenue) != 1 || realized.MonthlyRevenue[0].Month != "2026-09" {
		t.Fatalf("realized revenue must use the completion month: %+v", realized.MonthlyRevenue)
	}
	pipeline := buildRevenueDrilldown("pipeline", "all", nil, nil, jobs, nil, nil, nil)
	if len(pipeline.MonthlyRevenue) != 1 || pipeline.MonthlyRevenue[0].Month != "2026-08" {
		t.Fatalf("pipeline revenue must use the planned start month: %+v", pipeline.MonthlyRevenue)
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}
