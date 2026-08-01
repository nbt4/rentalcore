package pricing

import (
	"math"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
)

// PositionTotals contains the invoice totals calculated from a job's positions.
type PositionTotals struct {
	EventDays           int
	Subtotal            float64
	GlobalDiscount      float64
	Net                 float64
	TaxRate             float64
	Tax                 float64
	GrossBeforeDiscount float64
	Gross               float64
}

// CalculatePositionTotals is the single source of truth for position-based job totals.
func CalculatePositionTotals(job *models.Job, positions []models.JobPosition) PositionTotals {
	totals := PositionTotals{EventDays: eventDays(job.StartDate, job.EndDate)}
	taxRate := -1.0

	for _, position := range positions {
		dayFactor := 1.0
		if job.MultiplyByDays && totals.EventDays > 1 && position.FollowDayFactor > 0 {
			dayFactor = 1 + float64(totals.EventDays-1)*position.FollowDayFactor
		}

		amount := position.Quantity * position.UnitPrice * dayFactor
		discount := position.DiscountAmount + amount*position.DiscountPercent/100
		amount = math.Max(0, amount-discount)

		positionTaxRate := math.Max(0, position.TaxRate)
		if taxRate < 0 {
			taxRate = positionTaxRate
		} else if math.Abs(taxRate-positionTaxRate) > 0.0001 {
			taxRate = 0
		}

		if job.PricesIncludeTax {
			divisor := 1 + positionTaxRate/100
			net := amount
			if divisor > 0 {
				net = amount / divisor
			}
			totals.Subtotal += net
			totals.Tax += amount - net
		} else {
			totals.Subtotal += amount
			totals.Tax += amount * positionTaxRate / 100
		}
	}

	if taxRate < 0 {
		taxRate = 0
	}
	totals.TaxRate = taxRate
	totals.GrossBeforeDiscount = totals.Subtotal + totals.Tax

	discountFraction := 0.0
	if job.Discount > 0 {
		if strings.EqualFold(job.DiscountType, "percent") || strings.EqualFold(job.DiscountType, "percentage") {
			discountFraction = math.Min(job.Discount, 100) / 100
		} else if job.PricesIncludeTax {
			if totals.GrossBeforeDiscount > 0 {
				discountFraction = math.Min(job.Discount, totals.GrossBeforeDiscount) / totals.GrossBeforeDiscount
			}
		} else if totals.Subtotal > 0 {
			discountFraction = math.Min(job.Discount, totals.Subtotal) / totals.Subtotal
		}
	}

	taxBeforeDiscount := totals.Tax
	totals.GlobalDiscount = totals.Subtotal * discountFraction
	totals.Net = totals.Subtotal - totals.GlobalDiscount
	totals.Tax = taxBeforeDiscount * (1 - discountFraction)
	totals.Gross = totals.Net + totals.Tax

	totals.Subtotal = roundCurrency(totals.Subtotal)
	totals.GlobalDiscount = roundCurrency(totals.GlobalDiscount)
	totals.Net = roundCurrency(totals.Net)
	totals.Tax = roundCurrency(totals.Tax)
	totals.GrossBeforeDiscount = roundCurrency(totals.GrossBeforeDiscount)
	totals.Gross = roundCurrency(totals.Gross)
	return totals
}

func eventDays(start, end *time.Time) int {
	if start == nil || end == nil {
		return 1
	}
	days := int(end.Sub(*start).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

func roundCurrency(value float64) float64 {
	return math.Round(value*100) / 100
}
