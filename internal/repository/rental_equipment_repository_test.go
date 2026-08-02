package repository

import "testing"

func TestCalculateRentalTotalCostFollowsJobDaySetting(t *testing.T) {
	tests := []struct {
		name           string
		price          float64
		quantity       uint
		days           uint
		multiplyByDays bool
		want           float64
	}{
		{name: "flat event price", price: 474.74, quantity: 1, days: 3, multiplyByDays: false, want: 474.74},
		{name: "price per event day", price: 474.74, quantity: 1, days: 3, multiplyByDays: true, want: 1424.22},
		{name: "multiple units", price: 12.345, quantity: 2, days: 2, multiplyByDays: true, want: 49.38},
		{name: "zero days defaults to one", price: 25, quantity: 2, days: 0, multiplyByDays: true, want: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateRentalTotalCost(tt.price, tt.quantity, tt.days, tt.multiplyByDays); got != tt.want {
				t.Fatalf("got %.2f, want %.2f", got, tt.want)
			}
		})
	}
}
