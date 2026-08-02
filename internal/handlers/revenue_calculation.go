package handlers

import (
	"fmt"
	"math"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"

	"gorm.io/gorm"
)

type revenueAmounts struct {
	Net   float64
	Gross float64
}

func (a revenueAmounts) scale(factor float64) revenueAmounts {
	return revenueAmounts{Net: a.Net * factor, Gross: a.Gross * factor}
}

func splitInvoiceRevenue(invoiceAmount, taxRate float64, pricesIncludeTax bool) revenueAmounts {
	invoiceAmount = math.Max(invoiceAmount, 0)
	taxRate = math.Max(taxRate, 0)
	if taxRate == 0 {
		return revenueAmounts{Net: invoiceAmount, Gross: invoiceAmount}
	}

	taxFactor := 1 + taxRate/100
	if pricesIncludeTax {
		return revenueAmounts{Net: invoiceAmount / taxFactor, Gross: invoiceAmount}
	}
	return revenueAmounts{Net: invoiceAmount, Gross: invoiceAmount * taxFactor}
}

func positionEventDays(startDate, endDate *time.Time) int {
	if startDate == nil || endDate == nil {
		return 1
	}
	days := int(endDate.Sub(*startDate).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

func positionInvoiceRevenue(position models.JobPosition, multiplyByDays bool, eventDays int) float64 {
	dayFactor := 1.0
	if multiplyByDays && eventDays > 1 && position.FollowDayFactor > 0 {
		dayFactor = 1 + float64(eventDays-1)*position.FollowDayFactor
	}
	lineTotal := position.Quantity * position.UnitPrice * dayFactor
	discount := position.DiscountAmount + lineTotal*position.DiscountPercent/100
	return math.Max(lineTotal-discount, 0)
}

func jobDiscountFactor(subtotal, discount float64, discountType string) float64 {
	if subtotal <= 0 || discount <= 0 {
		return 1
	}
	discountAmount := discount
	if strings.EqualFold(discountType, "percent") || strings.EqualFold(discountType, "percentage") {
		discountAmount = subtotal * discount / 100
	}
	return math.Max(0, subtotal-discountAmount) / subtotal
}

func calculateJobPositionRevenue(job models.Job, positions []models.JobPosition) (float64, float64) {
	eventDays := positionEventDays(job.StartDate, job.EndDate)
	invoiceSubtotal := 0.0
	grossSubtotal := 0.0
	for _, position := range positions {
		invoiceRevenue := positionInvoiceRevenue(position, job.MultiplyByDays, eventDays)
		invoiceSubtotal += invoiceRevenue
		grossSubtotal += splitInvoiceRevenue(invoiceRevenue, position.TaxRate, job.PricesIncludeTax).Gross
	}
	finalRevenue := grossSubtotal * jobDiscountFactor(invoiceSubtotal, job.Discount, job.DiscountType)
	return roundAnalyticsMoney(grossSubtotal), roundAnalyticsMoney(finalRevenue)
}

func syncJobRevenue(db *gorm.DB, jobID uint) error {
	var job models.Job
	if err := db.Where("jobid = ?", jobID).First(&job).Error; err != nil {
		return fmt.Errorf("load job revenue settings: %w", err)
	}
	var positions []models.JobPosition
	if err := db.Where("job_id = ?", jobID).Find(&positions).Error; err != nil {
		return fmt.Errorf("load job positions: %w", err)
	}
	revenue, finalRevenue := calculateJobPositionRevenue(job, positions)
	if err := db.Model(&models.Job{}).Where("jobid = ?", jobID).Updates(map[string]interface{}{
		"revenue":       revenue,
		"final_revenue": finalRevenue,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("update job revenue: %w", err)
	}
	return nil
}

func syncJobRevenueIfPositions(db *gorm.DB, jobID uint) error {
	var count int64
	if err := db.Model(&models.JobPosition{}).Where("job_id = ?", jobID).Count(&count).Error; err != nil {
		return fmt.Errorf("count job positions: %w", err)
	}
	if count == 0 {
		return nil
	}
	return syncJobRevenue(db, jobID)
}
