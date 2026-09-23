package handlers

import (
	"testing"

	"go-barcode-webapp/internal/models"
)

func TestValidatePosition(t *testing.T) {
	productID := uint(7)
	tests := []struct {
		name string
		pos  models.JobPosition
		ok   bool
	}{
		{"valid product", models.JobPosition{PositionType: "product", ProductID: &productID, Quantity: 2, UnitPrice: 10}, true},
		{"fractional product", models.JobPosition{PositionType: "product", ProductID: &productID, Quantity: 1.5}, false},
		{"zero quantity", models.JobPosition{PositionType: "service", Quantity: 0}, false},
		{"negative price", models.JobPosition{PositionType: "service", Quantity: 1, UnitPrice: -1}, false},
		{"excess discount", models.JobPosition{PositionType: "service", Quantity: 1, DiscountPercent: 101}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePosition(&test.pos)
			if (err == nil) != test.ok {
				t.Fatalf("validatePosition() = %v, want success %v", err, test.ok)
			}
		})
	}
}
