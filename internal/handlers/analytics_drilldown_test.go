package handlers

import (
	"math"
	"testing"
	"time"
)

func TestBuildRevenueDrilldownReconcilesRevenueAndRentalMargin(t *testing.T) {
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	productID, rentalID, serviceID := uint(10), uint(20), uint(30)

	result := buildRevenueDrilldown(
		"all",
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
		"all",
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
	assertClose(t, result.AttributedRevenue, 250)
	assertClose(t, result.UnattributedRevenue, 50)
	assertClose(t, result.Categories[0].Revenue, 125)
	assertClose(t, result.Categories[2].Revenue, 125)
	assertClose(t, result.Categories[4].Revenue, 50)
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
		"all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 600, StartDate: &start, EndDate: &end, MultiplyByDays: false}},
		[]revenueDrilldownPosition{position}, nil, nil,
	)
	assertClose(t, flat.RentalCost, 474.74)

	perDay := buildRevenueDrilldown(
		"all", nil, nil,
		[]revenueDrilldownJob{{JobID: 1, Revenue: 600, StartDate: &start, EndDate: &end, MultiplyByDays: true}},
		[]revenueDrilldownPosition{position}, nil, nil,
	)
	assertClose(t, perDay.RentalCost, 1424.22)
}

func TestRevenueDrilldownPeriodRejectsUnknownValue(t *testing.T) {
	if _, _, err := revenueDrilldownPeriod("quarter", time.Now()); err == nil {
		t.Fatal("expected unsupported period to fail")
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}
