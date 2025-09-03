package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	db *gorm.DB
}

func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// Dashboard displays the main analytics dashboard
func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	currentUser, _ := GetCurrentUser(c)
	
	// Get period from query params (default: 30 days for better initial data)
	period := c.DefaultQuery("period", "30days")
	log.Printf("Analytics dashboard requested with period: %s", period)
	
	// Calculate date range
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "7days":
		startDate = endDate.AddDate(0, 0, -7)
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(0, 0, -30) // Default to 30 days
		period = "30days"
	}

	log.Printf("Analytics date range: %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	// Get analytics data with simplified approach
	analytics := h.getSimplifiedAnalyticsData(startDate, endDate)
	log.Printf("Analytics data retrieved for period %s", period)
	
	c.HTML(http.StatusOK, "analytics_dashboard_new.html", gin.H{
		"title":       "Analytics Dashboard",
		"currentPage": "analytics", 
		"user":        currentUser,
		"analytics":   analytics,
		"period":      period,
		"startDate":   startDate.Format("2006-01-02"),
		"endDate":     endDate.Format("2006-01-02"),
	})
}

// getSimplifiedAnalyticsData collects simplified analytics data for the new dashboard
func (h *AnalyticsHandler) getSimplifiedAnalyticsData(startDate, endDate time.Time) map[string]interface{} {
	log.Printf("Getting simplified analytics data from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	analytics := map[string]interface{}{
		"revenue":         h.getSimplifiedRevenue(startDate, endDate),
		"equipment":       h.getSimplifiedEquipment(startDate, endDate),
		"customers":       h.getSimplifiedCustomers(startDate, endDate),
		"jobs":            h.getSimplifiedJobs(startDate, endDate),
		"trends":          h.getSimplifiedTrends(startDate, endDate),
		"topEquipment":    h.getTopEquipment(startDate, endDate, 10),
		"topCustomers":    h.getTopCustomers(startDate, endDate, 10),
		"utilization":     h.getUtilizationMetrics(),
	}
	
	log.Printf("Simplified analytics data retrieved successfully")
	return analytics
}

// getSimplifiedRevenue calculates basic revenue metrics
func (h *AnalyticsHandler) getSimplifiedRevenue(startDate, endDate time.Time) map[string]interface{} {
	var totalRevenue float64
	var totalJobs int64
	
	// Simple query for total revenue from jobs in the period
	result := h.db.Raw(`
		SELECT 
			COALESCE(SUM(COALESCE(final_revenue, revenue, 0)), 0) as total_revenue,
			COUNT(*) as job_count
		FROM jobs 
		WHERE endDate BETWEEN ? AND ?
		AND (final_revenue > 0 OR revenue > 0)
	`, startDate, endDate).Row()
	
	result.Scan(&totalRevenue, &totalJobs)
	
	avgJobValue := float64(0)
	if totalJobs > 0 {
		avgJobValue = totalRevenue / float64(totalJobs)
	}
	
	log.Printf("Revenue data: %.2f total, %d jobs, %.2f avg", totalRevenue, totalJobs, avgJobValue)
	
	return map[string]interface{}{
		"totalRevenue": totalRevenue,
		"totalJobs":    totalJobs,
		"avgJobValue":  avgJobValue,
		"revenueGrowth": 0.0, // Simplified - no growth calculation
	}
}

// getSimplifiedEquipment calculates basic equipment metrics  
func (h *AnalyticsHandler) getSimplifiedEquipment(startDate, endDate time.Time) map[string]interface{} {
	var totalDevices, activeDevices int64
	
	// Count total devices
	h.db.Model(&models.Device{}).Count(&totalDevices)
	
	// Count active devices (checked out)
	h.db.Model(&models.Device{}).Where("status = ?", "checked out").Count(&activeDevices)
	
	utilizationRate := float64(0)
	if totalDevices > 0 {
		utilizationRate = (float64(activeDevices) / float64(totalDevices)) * 100
	}
	
	availableDevices := totalDevices - activeDevices
	
	log.Printf("Equipment data: %d total, %d active, %.1f%% utilization", totalDevices, activeDevices, utilizationRate)
	
	return map[string]interface{}{
		"totalDevices":     totalDevices,
		"activeDevices":    activeDevices,
		"availableDevices": availableDevices,
		"utilizationRate":  utilizationRate,
	}
}

// getSimplifiedCustomers calculates basic customer metrics
func (h *AnalyticsHandler) getSimplifiedCustomers(startDate, endDate time.Time) map[string]interface{} {
	var totalCustomers, activeCustomers int64
	
	// Count total customers
	h.db.Model(&models.Customer{}).Count(&totalCustomers)
	
	// Count active customers (with jobs in the period)
	h.db.Raw(`
		SELECT COUNT(DISTINCT customerID) 
		FROM jobs 
		WHERE startDate BETWEEN ? AND ?
	`, startDate, endDate).Scan(&activeCustomers)
	
	retentionRate := float64(0)
	if totalCustomers > 0 {
		retentionRate = (float64(activeCustomers) / float64(totalCustomers)) * 100
	}
	
	log.Printf("Customer data: %d total, %d active, %.1f%% retention", totalCustomers, activeCustomers, retentionRate)
	
	return map[string]interface{}{
		"totalCustomers":  totalCustomers,
		"activeCustomers": activeCustomers,
		"retentionRate":   retentionRate,
	}
}

// getSimplifiedJobs calculates basic job metrics
func (h *AnalyticsHandler) getSimplifiedJobs(startDate, endDate time.Time) map[string]interface{} {
	var completedJobs, activeJobs int64
	
	// Count completed jobs (statusID 3 or 4)
	h.db.Raw(`
		SELECT COUNT(*) 
		FROM jobs 
		WHERE endDate BETWEEN ? AND ? 
		AND statusID IN (3, 4)
	`, startDate, endDate).Scan(&completedJobs)
	
	// Count active jobs (statusID 1 or 2)
	h.db.Raw(`
		SELECT COUNT(*) 
		FROM jobs 
		WHERE startDate <= ? AND (endDate >= ? OR endDate IS NULL) 
		AND statusID IN (1, 2)
	`, endDate, startDate).Scan(&activeJobs)
	
	log.Printf("Job data: %d completed, %d active", completedJobs, activeJobs)
	
	return map[string]interface{}{
		"completedJobs": completedJobs,
		"activeJobs":    activeJobs,
		"totalJobs":     completedJobs + activeJobs,
	}
}

// getSimplifiedTrends provides basic trend data
func (h *AnalyticsHandler) getSimplifiedTrends(startDate, endDate time.Time) map[string]interface{} {
	// Simple daily revenue trend
	var trends []map[string]interface{}
	
	rows, err := h.db.Raw(`
		SELECT 
			DATE(endDate) as date,
			COALESCE(SUM(COALESCE(final_revenue, revenue, 0)), 0) as revenue,
			COUNT(*) as jobs
		FROM jobs
		WHERE endDate BETWEEN ? AND ?
		GROUP BY DATE(endDate)
		ORDER BY date
		LIMIT 30
	`, startDate, endDate).Rows()
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var revenue float64
			var jobs int
			
			rows.Scan(&date, &revenue, &jobs)
			trends = append(trends, map[string]interface{}{
				"date":    date,
				"revenue": revenue,
				"jobs":    jobs,
			})
		}
	}
	
	log.Printf("Trend data: %d data points", len(trends))
	
	return map[string]interface{}{
		"revenue": trends,
	}
}

// getAnalyticsData collects all analytics data for the dashboard
func (h *AnalyticsHandler) getAnalyticsData(startDate, endDate time.Time) map[string]interface{} {
	analytics := map[string]interface{}{
		"revenue":         h.getRevenueAnalytics(startDate, endDate),
		"equipment":       h.getEquipmentAnalytics(startDate, endDate),
		"customers":       h.getCustomerAnalytics(startDate, endDate),
		"jobs":           h.getJobAnalytics(startDate, endDate),
		"topEquipment":   h.getTopEquipment(startDate, endDate, 10),
		"topCustomers":   h.getTopCustomers(startDate, endDate, 10),
		"utilization":    h.getUtilizationMetrics(),
		"trends":         h.getTrendData(startDate, endDate),
	}
	
	// Ensure trends always has a proper structure
	if trends, ok := analytics["trends"].(map[string]interface{}); ok {
		if trends["revenue"] == nil {
			trends["revenue"] = []map[string]interface{}{}
		}
	} else {
		analytics["trends"] = map[string]interface{}{
			"revenue": []map[string]interface{}{},
		}
	}
	
	return analytics
}

// GetDeviceAnalytics returns detailed analytics for a specific device
func (h *AnalyticsHandler) GetDeviceAnalytics(c *gin.Context) {
	deviceID := c.Param("deviceId")
	
	// Get period from query params (default: all time)
	period := c.DefaultQuery("period", "all")
	
	// Calculate date range
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	case "all":
		startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // Far back date
	default:
		startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	analytics := h.getDeviceAnalyticsData(deviceID, startDate, endDate, period)
	
	c.JSON(http.StatusOK, analytics)
}

// getDeviceAnalyticsData collects detailed analytics for a specific device
func (h *AnalyticsHandler) getDeviceAnalyticsData(deviceID string, startDate, endDate time.Time, period string) map[string]interface{} {
	// Get device basic info
	var deviceInfo struct {
		DeviceID     string  `json:"deviceId" gorm:"column:deviceID"`
		ProductName  string  `json:"productName" gorm:"column:product_name"`
		SerialNumber *string `json:"serialNumber" gorm:"column:serial_number"`
		CategoryName string  `json:"categoryName" gorm:"column:category_name"`
		Status       string  `json:"status" gorm:"column:status"`
	}
	
	deviceResult := h.db.Raw(`
		SELECT 
			d.deviceID,
			COALESCE(p.name, 'Unknown Product') as product_name,
			d.serialnumber as serial_number,
			COALESCE(c.name, 'Unknown Category') as category_name,
			d.status
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN categories c ON p.categoryID = c.categoryID
		WHERE d.deviceID = ?
	`, deviceID).Scan(&deviceInfo)
	
	log.Printf("DEBUG: Device info query error: %v", deviceResult.Error)
	log.Printf("DEBUG: Device info - ID: %s, Name: %s, Serial: %v, Category: %s, Status: %s", 
		deviceInfo.DeviceID, deviceInfo.ProductName, deviceInfo.SerialNumber, deviceInfo.CategoryName, deviceInfo.Status)

	// Get total revenue and booking statistics
	var revenueStats struct {
		TotalRevenue  float64 `json:"totalRevenue"`
		TotalBookings int     `json:"totalBookings"`
		TotalRentals  int     `json:"totalRentals"`
		AvgDailyRate  float64 `json:"avgDailyRate"`
		FirstBooking  *time.Time `json:"firstBooking"`
		LastBooking   *time.Time `json:"lastBooking"`
	}
	
	h.db.Raw(`
		SELECT 
			COALESCE(SUM(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate) * (1 - j.discount/100)
							WHEN j.discount_type = 'amount' THEN 
								(jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate)) - j.discount
							ELSE 
								jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate)
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - LEAST(100, j.discount)/100)
							WHEN j.discount_type = 'amount' THEN 
								GREATEST(0, p.itemcostperday - j.discount)
							ELSE 
								p.itemcostperday
						END
				END
			), 0) as total_revenue,
			COUNT(DISTINCT j.jobID) as total_bookings,
			COUNT(DISTINCT j.jobID) as total_rentals,
			MIN(j.startDate) as first_booking,
			MAX(j.startDate) as last_booking
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID
		JOIN products p ON jd.deviceID = ? AND p.productID = (
			SELECT productID FROM devices WHERE deviceID = ?
		)
		WHERE jd.deviceID = ? 
		AND j.startDate BETWEEN ? AND ?
	`, deviceID, deviceID, deviceID, startDate, endDate).Scan(&revenueStats)
	
	// Calculate average daily rate
	if revenueStats.TotalRentals > 0 {
		revenueStats.AvgDailyRate = revenueStats.TotalRevenue / float64(revenueStats.TotalRentals)
	}

	// Get customer booking history with details - simplified approach
	type CustomerBooking struct {
		CustomerName  string    `json:"customer_name" gorm:"column:customer_name"`
		CustomerEmail *string   `json:"customer_email" gorm:"column:customer_email"`
		JobID         string    `json:"jobID" gorm:"column:jobID"`
		StartDate     time.Time `json:"startDate" gorm:"column:startDate"`
		EndDate       *time.Time `json:"endDate" gorm:"column:endDate"`
		Description   *string   `json:"description" gorm:"column:description"`
		RentalDays    int       `json:"rental_days" gorm:"column:rental_days"`
		Revenue       float64   `json:"revenue" gorm:"column:revenue"`
		DailyRate     float64   `json:"daily_rate" gorm:"column:daily_rate"`
		Discount      float64   `json:"discount" gorm:"column:discount"`
		DiscountType  *string   `json:"discount_type" gorm:"column:discount_type"`
		JobStatus     string    `json:"job_status" gorm:"column:job_status"`
	}
	
	var customerBookings []CustomerBooking
	
	// First try: Simple query to get any bookings for this device
	log.Printf("DEBUG: Looking for bookings for device: %s", deviceID)
	result := h.db.Raw(`
		SELECT 
			COALESCE(
				CASE 
					WHEN c.companyname IS NOT NULL AND c.companyname != '' THEN c.companyname
					WHEN c.firstname IS NOT NULL AND c.lastname IS NOT NULL THEN CONCAT(c.firstname, ' ', c.lastname)
					WHEN c.lastname IS NOT NULL THEN c.lastname
					WHEN c.firstname IS NOT NULL THEN c.firstname
					ELSE 'Unknown Customer'
				END
			) as customer_name,
			c.email as customer_email,
			j.jobID,
			j.startDate,
			j.endDate,
			j.description,
			GREATEST(1, CASE 
				WHEN j.endDate IS NOT NULL THEN DATEDIFF(j.endDate, j.startDate) + 1
				ELSE DATEDIFF(NOW(), j.startDate) + 1
			END) as rental_days,
			CASE 
				WHEN jd.custom_price IS NOT NULL THEN jd.custom_price
				ELSE COALESCE(p.itemcostperday, 0)
			END as daily_rate,
			COALESCE(j.discount, 0) as discount,
			j.discount_type,
			GREATEST(0, CASE 
				WHEN jd.custom_price IS NOT NULL THEN 
					CASE 
						WHEN j.discount_type = 'percent' AND j.discount > 0 THEN 
							jd.custom_price * (1 - LEAST(100, j.discount)/100)
						WHEN j.discount_type = 'amount' AND j.discount > 0 THEN 
							GREATEST(0, jd.custom_price - j.discount)
						ELSE 
							jd.custom_price
					END
				ELSE 
					CASE 
						WHEN j.discount_type = 'percent' AND j.discount > 0 THEN 
							COALESCE(p.itemcostperday, 0) * (1 - LEAST(100, j.discount)/100)
						WHEN j.discount_type = 'amount' AND j.discount > 0 THEN 
							GREATEST(0, COALESCE(p.itemcostperday, 0) - j.discount)
						ELSE 
							COALESCE(p.itemcostperday, 0)
					END
			END) as revenue,
			COALESCE(s.status, 'Unknown Status') as job_status
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID
		JOIN customers c ON j.customerID = c.customerID
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN status s ON j.statusID = s.statusID
		WHERE jd.deviceID = ?
		ORDER BY j.startDate DESC
		LIMIT 50
	`, deviceID).Scan(&customerBookings)
	
	log.Printf("DEBUG: Query result error: %v, found %d bookings", result.Error, len(customerBookings))
	log.Printf("DEBUG: Device ID requested: %s", deviceID)
	
	// Debug: print first booking details if any found
	if len(customerBookings) > 0 {
		first := customerBookings[0]
		log.Printf("DEBUG: First booking - Customer: %s, JobID: %s, Start: %v, End: %v, Days: %d, Rate: %.2f, Discount: %.2f (%s), Revenue: %.2f, Status: %s", 
			first.CustomerName, first.JobID, first.StartDate, first.EndDate, first.RentalDays, first.DailyRate, first.Discount, 
			*first.DiscountType, first.Revenue, first.JobStatus)
	}
	
	// If no bookings found, try even simpler query
	if len(customerBookings) == 0 {
		log.Printf("DEBUG: No bookings found, trying simpler query")
		h.db.Raw(`
			SELECT 
				COALESCE(
					CASE 
						WHEN c.companyname IS NOT NULL AND c.companyname != '' THEN c.companyname
						WHEN c.firstname IS NOT NULL AND c.lastname IS NOT NULL THEN CONCAT(c.firstname, ' ', c.lastname)
						WHEN c.lastname IS NOT NULL THEN c.lastname
						WHEN c.firstname IS NOT NULL THEN c.firstname
						ELSE 'Unknown Customer'
					END
				) as customer_name,
				c.email as customer_email,
				j.jobID,
				j.startDate,
				j.endDate,
				'Test Description' as description,
				3 as rental_days,
				100.0 as daily_rate,
				0.0 as discount,
				NULL as discount_type,
				300.0 as revenue,
				CAST(j.statusID as CHAR) as job_status
			FROM jobdevices jd
			JOIN jobs j ON jd.jobID = j.jobID
			JOIN customers c ON j.customerID = c.customerID
			WHERE jd.deviceID = ?
			LIMIT 5
		`, deviceID).Scan(&customerBookings)
		log.Printf("DEBUG: Simpler query found %d bookings", len(customerBookings))
		if len(customerBookings) > 0 {
			first := customerBookings[0]
			log.Printf("DEBUG: First booking from simpler query - Customer: %s, JobID: %s, Start: %v, End: %v, Days: %d, Rate: %.2f, Revenue: %.2f, Status: %s", 
				first.CustomerName, first.JobID, first.StartDate, first.EndDate, first.RentalDays, first.DailyRate, first.Revenue, first.JobStatus)
		}
	}

	// Get monthly revenue trend
	type MonthlyRevenue struct {
		Month    string  `json:"month"`
		Revenue  float64 `json:"revenue"`
		Bookings int     `json:"bookings"`
	}
	
	var monthlyRevenue []MonthlyRevenue
	h.db.Raw(`
		SELECT 
			DATE_FORMAT(j.startDate, '%Y-%m') as month,
			COALESCE(SUM(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate) * (1 - j.discount/100)
							WHEN j.discount_type = 'amount' THEN 
								(jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate)) - j.discount
							ELSE 
								jd.custom_price * DATEDIFF(COALESCE(j.endDate, NOW()), j.startDate)
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - LEAST(100, j.discount)/100)
							WHEN j.discount_type = 'amount' THEN 
								GREATEST(0, p.itemcostperday - j.discount)
							ELSE 
								p.itemcostperday
						END
				END
			), 0) as revenue,
			COUNT(DISTINCT j.jobID) as bookings
		FROM jobdevices jd
		JOIN jobs j ON jd.jobID = j.jobID
		JOIN devices d ON jd.deviceID = d.deviceID
		JOIN products p ON d.productID = p.productID
		WHERE jd.deviceID = ? 
		AND j.startDate BETWEEN ? AND ?
		GROUP BY DATE_FORMAT(j.startDate, '%Y-%m')
		ORDER BY month DESC
		LIMIT 12
	`, deviceID, startDate, endDate).Scan(&monthlyRevenue)

	// Get utilization metrics
	var utilizationStats struct {
		DaysBooked     int     `json:"daysBooked"`
		DaysAvailable  int     `json:"daysAvailable"`
		UtilizationRate float64 `json:"utilizationRate"`
	}
	
	totalDaysInPeriod := int(endDate.Sub(startDate).Hours() / 24)
	utilizationStats.DaysAvailable = totalDaysInPeriod
	utilizationStats.DaysBooked = revenueStats.TotalRentals
	
	if totalDaysInPeriod > 0 {
		utilizationStats.UtilizationRate = (float64(revenueStats.TotalRentals) / float64(totalDaysInPeriod)) * 100
	}

	// Transform data for frontend compatibility
	var bookingCount int = revenueStats.TotalBookings
	var avgDuration float64
	var avgBookingValue float64 = revenueStats.AvgDailyRate
	
	if len(customerBookings) > 0 {
		totalDays := 0
		for _, booking := range customerBookings {
			totalDays += booking.RentalDays
		}
		avgDuration = float64(totalDays) / float64(len(customerBookings))
	}
	
	// Transform monthly revenue to daily trends for charts
	var revenueTrends []map[string]interface{}
	for _, monthly := range monthlyRevenue {
		revenueTrends = append(revenueTrends, map[string]interface{}{
			"date":    monthly.Month + "-01", // Add day for proper date parsing
			"revenue": monthly.Revenue,
			"jobs":    monthly.Bookings,
		})
	}
	
	// Create status distribution based on booking data
	statusCounts := map[string]int{
		"Completed": 0,
		"Active":    0,
		"Cancelled": 0,
	}
	
	for _, booking := range customerBookings {
		status := booking.JobStatus
		if status == "completed" || status == "finished" {
			statusCounts["Completed"]++
		} else if status == "active" || status == "ongoing" {
			statusCounts["Active"]++
		} else if status == "cancelled" {
			statusCounts["Cancelled"]++
		} else {
			statusCounts["Completed"]++ // Default unknown status to completed
		}
	}
	
	// Transform customer bookings for frontend
	var transformedBookings []map[string]interface{}
	for _, booking := range customerBookings {
		transformedBookings = append(transformedBookings, map[string]interface{}{
			"jobID":        booking.JobID,
			"customerName": booking.CustomerName,
			"startDate":    booking.StartDate.Format("2006-01-02"),
			"endDate":      func() string {
				if booking.EndDate != nil {
					return booking.EndDate.Format("2006-01-02")
				}
				return time.Now().Format("2006-01-02")
			}(),
			"revenue": booking.Revenue,
			"status":  booking.JobStatus,
		})
	}
	
	return map[string]interface{}{
		"device": map[string]interface{}{
			"deviceId":     deviceInfo.DeviceID,
			"productName":  deviceInfo.ProductName,
			"serialNumber": deviceInfo.SerialNumber,
			"categoryName": deviceInfo.CategoryName,
			"status":       deviceInfo.Status,
		},
		"revenue": map[string]interface{}{
			"totalRevenue":     revenueStats.TotalRevenue,
			"bookingCount":     bookingCount,
			"avgDuration":      avgDuration,
			"avgBookingValue":  avgBookingValue,
			"utilizationRate":  utilizationStats.UtilizationRate,
		},
		"trends": map[string]interface{}{
			"revenue": revenueTrends,
		},
		"statusDistribution": map[string]interface{}{
			"labels": []string{"Completed", "Active", "Cancelled"},
			"data":   []int{statusCounts["Completed"], statusCounts["Active"], statusCounts["Cancelled"]},
		},
		"bookings": transformedBookings,
		"period":   period,
		"date_range": map[string]interface{}{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	}
}

// getRevenueAnalytics calculates revenue metrics
func (h *AnalyticsHandler) getRevenueAnalytics(startDate, endDate time.Time) map[string]interface{} {
	var totalRevenue, avgJobValue float64
	var totalJobs int64

	// Total revenue and job count - try different revenue fields
	// First try final_revenue
	h.db.Model(&models.Job{}).
		Where("endDate BETWEEN ? AND ? AND final_revenue IS NOT NULL AND final_revenue > 0", startDate, endDate).
		Select("COALESCE(SUM(final_revenue), 0) as total, COUNT(*) as count, COALESCE(AVG(final_revenue), 0) as avg").
		Row().Scan(&totalRevenue, &totalJobs, &avgJobValue)
	
	// If no final_revenue data, try regular revenue field
	if totalRevenue == 0 {
		h.db.Model(&models.Job{}).
			Where("endDate BETWEEN ? AND ? AND revenue IS NOT NULL AND revenue > 0", startDate, endDate).
			Select("COALESCE(SUM(revenue), 0) as total, COUNT(*) as count, COALESCE(AVG(revenue), 0) as avg").
			Row().Scan(&totalRevenue, &totalJobs, &avgJobValue)
	}

	// Previous period for comparison
	prevStartDate := startDate.AddDate(0, 0, -int(endDate.Sub(startDate).Hours()/24))
	prevEndDate := startDate
	
	var prevRevenue float64
	var prevJobs int64
	// Use the same flexible approach for previous period
	h.db.Model(&models.Job{}).
		Where("endDate BETWEEN ? AND ? AND final_revenue IS NOT NULL AND final_revenue > 0", prevStartDate, prevEndDate).
		Select("COALESCE(SUM(final_revenue), 0) as total, COUNT(*) as count").
		Row().Scan(&prevRevenue, &prevJobs)
	
	if prevRevenue == 0 {
		h.db.Model(&models.Job{}).
			Where("endDate BETWEEN ? AND ? AND revenue IS NOT NULL AND revenue > 0", prevStartDate, prevEndDate).
			Select("COALESCE(SUM(revenue), 0) as total, COUNT(*) as count").
			Row().Scan(&prevRevenue, &prevJobs)
	}

	// Calculate growth rates
	revenueGrowth := float64(0)
	if prevRevenue > 0 {
		revenueGrowth = ((totalRevenue - prevRevenue) / prevRevenue) * 100
	}

	jobsGrowth := float64(0)
	if prevJobs > 0 {
		jobsGrowth = ((float64(totalJobs) - float64(prevJobs)) / float64(prevJobs)) * 100
	}

	return map[string]interface{}{
		"totalRevenue":   totalRevenue,
		"totalJobs":      totalJobs,
		"avgJobValue":    avgJobValue,
		"revenueGrowth":  revenueGrowth,
		"jobsGrowth":     jobsGrowth,
	}
}

// getEquipmentAnalytics calculates equipment metrics
func (h *AnalyticsHandler) getEquipmentAnalytics(startDate, endDate time.Time) map[string]interface{} {
	var totalDevices, activeDevices, maintenanceDevices int64

	// Total devices
	h.db.Model(&models.Device{}).Count(&totalDevices)

	// Active devices (assigned to jobs)
	h.db.Model(&models.Device{}).Where("status IN (?)", []string{"checked out"}).Count(&activeDevices)

	// Devices in maintenance
	h.db.Model(&models.Device{}).Where("status = ?", "maintance").Count(&maintenanceDevices)

	// Utilization rate
	utilizationRate := float64(0)
	if totalDevices > 0 {
		utilizationRate = (float64(activeDevices) / float64(totalDevices)) * 100
	}

	// Revenue per device - calculate individual device revenue with discount applied
	var totalDeviceRevenue float64
	h.db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN jd.custom_price IS NOT NULL THEN 
					CASE 
						WHEN j.discount_type = 'percent' THEN 
							jd.custom_price * (1 - j.discount/100)
						ELSE 
							jd.custom_price * (1 - (j.discount / NULLIF(j.revenue, 0)))
					END
				ELSE 
					CASE 
						WHEN j.discount_type = 'percent' THEN 
							p.itemcostperday * (1 - j.discount/100)
						ELSE 
							p.itemcostperday * (1 - (j.discount / NULLIF(j.revenue, 0)))
					END
			END
		), 0)
		FROM jobs j
		INNER JOIN jobdevices jd ON j.jobID = jd.jobID
		INNER JOIN devices d ON jd.deviceID = d.deviceID
		INNER JOIN products p ON d.productID = p.productID
		WHERE j.endDate BETWEEN ? AND ?
	`, startDate, endDate).Scan(&totalDeviceRevenue)

	revenuePerDevice := float64(0)
	if totalDevices > 0 {
		revenuePerDevice = totalDeviceRevenue / float64(totalDevices)
	}

	return map[string]interface{}{
		"totalDevices":      totalDevices,
		"activeDevices":     activeDevices,
		"maintenanceDevices": maintenanceDevices,
		"utilizationRate":   utilizationRate,
		"revenuePerDevice":  revenuePerDevice,
		"availableDevices":  totalDevices - activeDevices - maintenanceDevices,
	}
}

// getCustomerAnalytics calculates customer metrics
func (h *AnalyticsHandler) getCustomerAnalytics(startDate, endDate time.Time) map[string]interface{} {
	var totalCustomers, activeCustomers, newCustomers int64

	// Total customers
	h.db.Model(&models.Customer{}).Count(&totalCustomers)

	// Active customers (with jobs in period)
	h.db.Model(&models.Customer{}).
		Joins("INNER JOIN jobs ON customers.customerID = jobs.customerID").
		Where("jobs.startDate BETWEEN ? AND ?", startDate, endDate).
		Distinct("customers.customerID").
		Count(&activeCustomers)

	// New customers in period
	h.db.Model(&models.Customer{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&newCustomers)

	// Customer retention rate
	retentionRate := float64(0)
	if totalCustomers > 0 {
		retentionRate = (float64(activeCustomers) / float64(totalCustomers)) * 100
	}

	return map[string]interface{}{
		"totalCustomers":  totalCustomers,
		"activeCustomers": activeCustomers,
		"newCustomers":    newCustomers,
		"retentionRate":   retentionRate,
	}
}

// getJobAnalytics calculates job metrics
func (h *AnalyticsHandler) getJobAnalytics(startDate, endDate time.Time) map[string]interface{} {
	var completedJobs, activeJobs, overdueJobs int64
	var avgJobDuration float64

	// Completed jobs
	h.db.Model(&models.Job{}).
		Where("endDate BETWEEN ? AND ? AND statusID IN (?)", startDate, endDate, []int{3, 4}).
		Count(&completedJobs)

	// Active jobs
	h.db.Model(&models.Job{}).
		Where("startDate <= ? AND (endDate >= ? OR endDate IS NULL) AND statusID IN (?)", 
			endDate, startDate, []int{1, 2}).
		Count(&activeJobs)

	// Overdue jobs
	h.db.Model(&models.Job{}).
		Where("endDate < ? AND statusID NOT IN (?)", time.Now(), []int{3, 4}).
		Count(&overdueJobs)

	// Average job duration
	h.db.Model(&models.Job{}).
		Where("endDate BETWEEN ? AND ? AND startDate IS NOT NULL AND endDate IS NOT NULL", 
			startDate, endDate).
		Select("AVG(DATEDIFF(endDate, startDate))").
		Scan(&avgJobDuration)

	return map[string]interface{}{
		"completedJobs":    completedJobs,
		"activeJobs":       activeJobs,
		"overdueJobs":      overdueJobs,
		"avgJobDuration":   avgJobDuration,
	}
}

// getTopEquipment returns top performing equipment
func (h *AnalyticsHandler) getTopEquipment(startDate, endDate time.Time, limit int) []map[string]interface{} {
	var results []map[string]interface{}

	rows, err := h.db.Raw(`
		SELECT 
			d.deviceID,
			p.name as product_name,
			COUNT(jd.jobID) as rental_count,
			COALESCE(SUM(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * (1 - j.discount/100)
							ELSE 
								jd.custom_price * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - j.discount/100)
							ELSE 
								p.itemcostperday * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
				END
			), 0) as total_revenue,
			COALESCE(AVG(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * (1 - j.discount/100)
							ELSE 
								jd.custom_price * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - j.discount/100)
							ELSE 
								p.itemcostperday * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
				END
			), 0) as avg_revenue
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN jobdevices jd ON d.deviceID = jd.deviceID
		LEFT JOIN jobs j ON jd.jobID = j.jobID AND j.endDate BETWEEN ? AND ? 		GROUP BY d.deviceID, p.name
		ORDER BY total_revenue DESC
		LIMIT ?
	`, startDate, endDate, limit).Rows()

	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var deviceID, productName string
		var rentalCount int
		var totalRevenue, avgRevenue float64

		rows.Scan(&deviceID, &productName, &rentalCount, &totalRevenue, &avgRevenue)
		
		results = append(results, map[string]interface{}{
			"deviceID":     deviceID,
			"productName":  productName,
			"rentalCount":  rentalCount,
			"totalRevenue": totalRevenue,
			"avgRevenue":   avgRevenue,
		})
	}

	return results
}

// getAllDeviceRevenues returns revenue data for ALL devices (not limited)
func (h *AnalyticsHandler) getAllDeviceRevenues(startDate, endDate time.Time, sortColumn, order string) []map[string]interface{} {
	var results []map[string]interface{}

	// Build the query with dynamic sorting
	query := `
		SELECT 
			d.deviceID,
			p.name as product_name,
			COUNT(jd.jobID) as rental_count,
			COALESCE(SUM(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * (1 - j.discount/100)
							ELSE 
								jd.custom_price * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - j.discount/100)
							ELSE 
								p.itemcostperday * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
				END
			), 0) as total_revenue,
			COALESCE(AVG(
				CASE 
					WHEN jd.custom_price IS NOT NULL THEN 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								jd.custom_price * (1 - j.discount/100)
							ELSE 
								jd.custom_price * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
					ELSE 
						CASE 
							WHEN j.discount_type = 'percent' THEN 
								p.itemcostperday * (1 - j.discount/100)
							ELSE 
								p.itemcostperday * (1 - (j.discount / NULLIF(j.revenue, 0)))
						END
				END
			), 0) as avg_revenue,
			p.itemcostperday as product_price,
			d.status as device_status
		FROM devices d
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN jobdevices jd ON d.deviceID = jd.deviceID
		LEFT JOIN jobs j ON jd.jobID = j.jobID AND j.endDate BETWEEN ? AND ? 		GROUP BY d.deviceID, p.name, p.itemcostperday, d.status
		ORDER BY ` + sortColumn + ` ` + order

	rows, err := h.db.Raw(query, startDate, endDate).Rows()
	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var deviceID, productName, deviceStatus string
		var rentalCount int
		var totalRevenue, avgRevenue, productPrice float64

		rows.Scan(&deviceID, &productName, &rentalCount, &totalRevenue, &avgRevenue, &productPrice, &deviceStatus)
		
		results = append(results, map[string]interface{}{
			"deviceID":     deviceID,
			"productName":  productName,
			"rentalCount":  rentalCount,
			"totalRevenue": totalRevenue,
			"avgRevenue":   avgRevenue,
			"productPrice": productPrice,
			"deviceStatus": deviceStatus,
		})
	}

	return results
}

// getTopCustomers returns top customers by revenue
func (h *AnalyticsHandler) getTopCustomers(startDate, endDate time.Time, limit int) []map[string]interface{} {
	var results []map[string]interface{}

	rows, err := h.db.Raw(`
		SELECT 
			c.customerID,
			COALESCE(c.companyname, CONCAT(c.firstname, ' ', c.lastname)) as customer_name,
			COUNT(j.jobID) as job_count,
			COALESCE(SUM(j.final_revenue), 0) as total_revenue,
			COALESCE(AVG(j.final_revenue), 0) as avg_revenue
		FROM customers c
		LEFT JOIN jobs j ON c.customerID = j.customerID 
			AND j.endDate BETWEEN ? AND ?
			AND j.final_revenue IS NOT NULL
					WHERE c.customerID IS NOT NULL
		GROUP BY c.customerID, c.companyname, c.firstname, c.lastname
		HAVING total_revenue > 0
		ORDER BY total_revenue DESC
		LIMIT ?
	`, startDate, endDate, limit).Rows()

	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var customerID int
		var customerName string
		var jobCount int
		var totalRevenue, avgRevenue float64

		rows.Scan(&customerID, &customerName, &jobCount, &totalRevenue, &avgRevenue)
		
		results = append(results, map[string]interface{}{
			"customerID":   customerID,
			"customerName": customerName,
			"jobCount":     jobCount,
			"totalRevenue": totalRevenue,
			"avgRevenue":   avgRevenue,
		})
	}

	return results
}

// getUtilizationMetrics calculates equipment utilization rates
func (h *AnalyticsHandler) getUtilizationMetrics() map[string]interface{} {
	var results []map[string]interface{}

	rows, err := h.db.Raw(`
		SELECT 
			p.name as product_name,
			COUNT(d.deviceID) as total_devices,
			SUM(CASE WHEN d.status = 'checked out' THEN 1 ELSE 0 END) as active_devices,
			ROUND((SUM(CASE WHEN d.status = 'checked out' THEN 1 ELSE 0 END) * 100.0) / COUNT(d.deviceID), 2) as utilization_rate
		FROM products p
		LEFT JOIN devices d ON p.productID = d.productID
		GROUP BY p.productID, p.name
		HAVING COUNT(d.deviceID) > 0
		ORDER BY utilization_rate DESC
	`).Rows()

	if err != nil {
		return map[string]interface{}{"categories": results}
	}
	defer rows.Close()

	for rows.Next() {
		var productName string
		var totalDevices, activeDevices int
		var utilizationRate float64

		rows.Scan(&productName, &totalDevices, &activeDevices, &utilizationRate)
		
		results = append(results, map[string]interface{}{
			"productName":     productName,
			"totalDevices":    totalDevices,
			"activeDevices":   activeDevices,
			"utilizationRate": utilizationRate,
		})
	}

	return map[string]interface{}{
		"categories": results,
	}
}

// getTrendData returns daily/weekly trend data for charts
func (h *AnalyticsHandler) getTrendData(startDate, endDate time.Time) map[string]interface{} {
	// Daily revenue trend
	revenueRows, err := h.db.Raw(`
		SELECT 
			DATE(j.endDate) as date,
			COALESCE(SUM(j.final_revenue), 0) as revenue,
			COUNT(j.jobID) as jobs
		FROM jobs j
		WHERE j.endDate BETWEEN ? AND ?
		GROUP BY DATE(j.endDate)
		ORDER BY date
	`, startDate, endDate).Rows()

	var revenueTrend []map[string]interface{}
	if err == nil {
		defer revenueRows.Close()
		for revenueRows.Next() {
			var date time.Time
			var revenue float64
			var jobs int

			revenueRows.Scan(&date, &revenue, &jobs)
			revenueTrend = append(revenueTrend, map[string]interface{}{
				"date":    date.Format("2006-01-02"),
				"revenue": revenue,
				"jobs":    jobs,
			})
		}
	}

	return map[string]interface{}{
		"revenue": revenueTrend,
	}
}

// GetRevenueAPI returns revenue data as JSON API
func (h *AnalyticsHandler) GetRevenueAPI(c *gin.Context) {
	period := c.DefaultQuery("period", "1year")
	
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "7days":
		startDate = endDate.AddDate(0, 0, -7)
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(-1, 0, 0)
	}

	analytics := h.getRevenueAnalytics(startDate, endDate)
	c.JSON(http.StatusOK, analytics)
}

// GetEquipmentAPI returns equipment analytics as JSON API
func (h *AnalyticsHandler) GetEquipmentAPI(c *gin.Context) {
	period := c.DefaultQuery("period", "1year")
	
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "7days":
		startDate = endDate.AddDate(0, 0, -7)
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(-1, 0, 0)
	}

	analytics := h.getEquipmentAnalytics(startDate, endDate)
	c.JSON(http.StatusOK, analytics)
}

// GetAllDeviceRevenuesAPI returns revenue data for ALL devices as JSON API
func (h *AnalyticsHandler) GetAllDeviceRevenuesAPI(c *gin.Context) {
	period := c.DefaultQuery("period", "1year")
	sortBy := c.DefaultQuery("sort", "revenue") // revenue, device_id, product_name, rental_count
	order := c.DefaultQuery("order", "desc")    // asc, desc
	
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "7days":
		startDate = endDate.AddDate(0, 0, -7)
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(-1, 0, 0)
	}

	// Validate sort and order parameters
	validSorts := map[string]string{
		"revenue":      "total_revenue",
		"device_id":    "d.deviceID",
		"product_name": "p.name",
		"rental_count": "rental_count",
	}
	
	sortColumn, exists := validSorts[sortBy]
	if !exists {
		sortColumn = "total_revenue"
	}
	
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	allDevices := h.getAllDeviceRevenues(startDate, endDate, sortColumn, order)
	c.JSON(http.StatusOK, gin.H{
		"devices": allDevices,
		"period":  period,
		"count":   len(allDevices),
	})
}

// ExportAnalytics exports analytics data to CSV/Excel
func (h *AnalyticsHandler) ExportAnalytics(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	period := c.DefaultQuery("period", "1year")
	
	endDate := time.Now()
	var startDate time.Time
	
	switch period {
	case "7days":
		startDate = endDate.AddDate(0, 0, -7)
	case "30days":
		startDate = endDate.AddDate(0, 0, -30)
	case "90days":
		startDate = endDate.AddDate(0, 0, -90)
	case "1year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(-1, 0, 0)
	}

	if format == "csv" {
		h.exportToCSV(c, startDate, endDate)
	} else if format == "pdf" {
		h.exportToPDF(c, startDate, endDate)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format"})
	}
}

// exportToCSV exports analytics data to CSV format
func (h *AnalyticsHandler) exportToCSV(c *gin.Context, startDate, endDate time.Time) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="analytics_`+time.Now().Format("2006-01-02")+`.csv"`)

	// Get analytics data
	analytics := h.getAnalyticsData(startDate, endDate)
	
	// Write CSV headers and data
	csvData := "Metric,Value\n"
	
	// Revenue metrics
	if revenue, ok := analytics["revenue"].(map[string]interface{}); ok {
		csvData += "Total Revenue," + strconv.FormatFloat(revenue["totalRevenue"].(float64), 'f', 2, 64) + "\n"
		csvData += "Total Jobs," + strconv.FormatInt(revenue["totalJobs"].(int64), 10) + "\n"
		csvData += "Average Job Value," + strconv.FormatFloat(revenue["avgJobValue"].(float64), 'f', 2, 64) + "\n"
		if growth, ok := revenue["revenueGrowth"].(float64); ok {
			csvData += "Revenue Growth %," + strconv.FormatFloat(growth, 'f', 1, 64) + "\n"
		}
	}
	
	// Equipment metrics
	if equipment, ok := analytics["equipment"].(map[string]interface{}); ok {
		csvData += "Total Devices," + strconv.FormatInt(equipment["totalDevices"].(int64), 10) + "\n"
		csvData += "Active Devices," + strconv.FormatInt(equipment["activeDevices"].(int64), 10) + "\n"
		csvData += "Utilization Rate %," + strconv.FormatFloat(equipment["utilizationRate"].(float64), 'f', 1, 64) + "\n"
		if revenue, ok := equipment["revenuePerDevice"].(float64); ok {
			csvData += "Revenue per Device," + strconv.FormatFloat(revenue, 'f', 2, 64) + "\n"
		}
	}
	
	// Customer metrics
	if customers, ok := analytics["customers"].(map[string]interface{}); ok {
		csvData += "Total Customers," + strconv.FormatInt(customers["totalCustomers"].(int64), 10) + "\n"
		csvData += "Active Customers," + strconv.FormatInt(customers["activeCustomers"].(int64), 10) + "\n"
		if retention, ok := customers["retentionRate"].(float64); ok {
			csvData += "Customer Retention %," + strconv.FormatFloat(retention, 'f', 1, 64) + "\n"
		}
	}
	
	// Top equipment section
	csvData += "\nTop Equipment by Revenue\n"
	csvData += "Device ID,Product Name,Rental Count,Total Revenue\n"
	if topEquipment, ok := analytics["topEquipment"].([]map[string]interface{}); ok {
		for _, equipment := range topEquipment {
			csvData += fmt.Sprintf("%s,%s,%v,%.2f\n",
				equipment["deviceID"].(string),
				equipment["productName"].(string),
				equipment["rentalCount"],
				equipment["totalRevenue"].(float64),
			)
		}
	}
	
	// Top customers section
	csvData += "\nTop Customers by Revenue\n"
	csvData += "Customer Name,Job Count,Total Revenue\n"
	if topCustomers, ok := analytics["topCustomers"].([]map[string]interface{}); ok {
		for _, customer := range topCustomers {
			csvData += fmt.Sprintf("%s,%v,%.2f\n",
				customer["customerName"].(string),
				customer["jobCount"],
				customer["totalRevenue"].(float64),
			)
		}
	}

	c.String(http.StatusOK, csvData)
}

// exportToPDF exports analytics data to PDF format
func (h *AnalyticsHandler) exportToPDF(c *gin.Context, startDate, endDate time.Time) {
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="analytics_`+time.Now().Format("2006-01-02")+`.pdf"`)

	// Get analytics data
	analytics := h.getAnalyticsData(startDate, endDate)

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Add title
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(30, 64, 175) // Blue color
	pdf.Cell(190, 15, "Analytics Report")
	pdf.Ln(20)

	// Add date range
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(75, 85, 99) // Gray color
	pdf.Cell(190, 8, fmt.Sprintf("Period: %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))
	pdf.Ln(15)

	// Revenue Section
	h.addPDFSection(pdf, "Revenue Analytics", analytics["revenue"])

	// Equipment Section
	h.addPDFSection(pdf, "Equipment Analytics", analytics["equipment"])

	// Customer Section
	h.addPDFSection(pdf, "Customer Analytics", analytics["customers"])

	// Job Section
	h.addPDFSection(pdf, "Job Analytics", analytics["jobs"])

	// Top Equipment Table
	h.addTopEquipmentTable(pdf, analytics["topEquipment"])

	// Top Customers Table
	h.addTopCustomersTable(pdf, analytics["topCustomers"])

	// Output PDF
	err := pdf.Output(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}
}

// addPDFSection adds a section to the PDF with key metrics
func (h *AnalyticsHandler) addPDFSection(pdf *gofpdf.Fpdf, title string, data interface{}) {
	// Section title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.Cell(190, 10, title)
	pdf.Ln(12)

	// Create a colored background rectangle
	pdf.SetFillColor(248, 250, 252) // Light gray background
	pdf.Rect(10, pdf.GetY()-2, 190, 25, "F")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(75, 85, 99)

	if dataMap, ok := data.(map[string]interface{}); ok {
		switch title {
		case "Revenue Analytics":
			h.addRevenueMetrics(pdf, dataMap)
		case "Equipment Analytics":
			h.addEquipmentMetrics(pdf, dataMap)
		case "Customer Analytics":
			h.addCustomerMetrics(pdf, dataMap)
		case "Job Analytics":
			h.addJobMetrics(pdf, dataMap)
		}
	}

	pdf.Ln(15)
}

// addRevenueMetrics adds revenue metrics to PDF
func (h *AnalyticsHandler) addRevenueMetrics(pdf *gofpdf.Fpdf, data map[string]interface{}) {
	y := pdf.GetY()
	
	// Total Revenue
	if totalRevenue, ok := data["totalRevenue"].(float64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Total Revenue: EUR %.2f", totalRevenue))
	}
	
	// Total Jobs
	if totalJobs, ok := data["totalJobs"].(int64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Total Jobs: %d", totalJobs))
	}

	y += 8
	pdf.SetXY(15, y)
	
	// Average Job Value
	if avgJobValue, ok := data["avgJobValue"].(float64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Average Job Value: EUR %.2f", avgJobValue))
	}
	
	// Revenue Growth
	if revenueGrowth, ok := data["revenueGrowth"].(float64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Revenue Growth: %.1f%%", revenueGrowth))
	}
}

// addEquipmentMetrics adds equipment metrics to PDF
func (h *AnalyticsHandler) addEquipmentMetrics(pdf *gofpdf.Fpdf, data map[string]interface{}) {
	y := pdf.GetY()
	
	// Total Devices
	if totalDevices, ok := data["totalDevices"].(int64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Total Devices: %d", totalDevices))
	}
	
	// Active Devices
	if activeDevices, ok := data["activeDevices"].(int64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Active Devices: %d", activeDevices))
	}

	y += 8
	pdf.SetXY(15, y)
	
	// Utilization Rate
	if utilizationRate, ok := data["utilizationRate"].(float64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Utilization Rate: %.1f%%", utilizationRate))
	}
	
	// Revenue per Device
	if revenuePerDevice, ok := data["revenuePerDevice"].(float64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Revenue per Device: EUR %.2f", revenuePerDevice))
	}
}

// addCustomerMetrics adds customer metrics to PDF
func (h *AnalyticsHandler) addCustomerMetrics(pdf *gofpdf.Fpdf, data map[string]interface{}) {
	y := pdf.GetY()
	
	// Total Customers
	if totalCustomers, ok := data["totalCustomers"].(int64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Total Customers: %d", totalCustomers))
	}
	
	// Active Customers
	if activeCustomers, ok := data["activeCustomers"].(int64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Active Customers: %d", activeCustomers))
	}

	y += 8
	pdf.SetXY(15, y)
	
	// New Customers
	if newCustomers, ok := data["newCustomers"].(int64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("New Customers: %d", newCustomers))
	}
	
	// Retention Rate
	if retentionRate, ok := data["retentionRate"].(float64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Retention Rate: %.1f%%", retentionRate))
	}
}

// addJobMetrics adds job metrics to PDF
func (h *AnalyticsHandler) addJobMetrics(pdf *gofpdf.Fpdf, data map[string]interface{}) {
	y := pdf.GetY()
	
	// Completed Jobs
	if completedJobs, ok := data["completedJobs"].(int64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Completed Jobs: %d", completedJobs))
	}
	
	// Active Jobs
	if activeJobs, ok := data["activeJobs"].(int64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Active Jobs: %d", activeJobs))
	}

	y += 8
	pdf.SetXY(15, y)
	
	// Overdue Jobs
	if overdueJobs, ok := data["overdueJobs"].(int64); ok {
		pdf.Cell(90, 6, fmt.Sprintf("Overdue Jobs: %d", overdueJobs))
	}
	
	// Average Duration
	if avgJobDuration, ok := data["avgJobDuration"].(float64); ok {
		pdf.SetXY(105, y)
		pdf.Cell(90, 6, fmt.Sprintf("Avg Duration: %.1f days", avgJobDuration))
	}
}

// addTopEquipmentTable adds top equipment table to PDF
func (h *AnalyticsHandler) addTopEquipmentTable(pdf *gofpdf.Fpdf, data interface{}) {
	if pdf.GetY() > 220 {
		pdf.AddPage()
	}

	// Table title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.Cell(190, 10, "Top Equipment by Revenue")
	pdf.Ln(12)

	// Table headers
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(40, 8, "Device ID", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 8, "Product Name", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Rentals", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Revenue", "1", 0, "C", true, 0, "")
	pdf.Ln(8)

	// Table data
	pdf.SetFont("Arial", "", 8)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetTextColor(51, 51, 51)

	if equipmentList, ok := data.([]map[string]interface{}); ok {
		for i, equipment := range equipmentList {
			if i >= 10 { // Limit to top 10
				break
			}
			
			// Alternate row colors
			if i%2 == 1 {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			deviceID := ""
			if id, ok := equipment["deviceID"].(string); ok {
				deviceID = id
			}

			productName := ""
			if name, ok := equipment["productName"].(string); ok {
				productName = name
				if len(productName) > 25 {
					productName = productName[:22] + "..."
				}
			}

			rentalCount := "0"
			if count, ok := equipment["rentalCount"].(int); ok {
				rentalCount = strconv.Itoa(count)
			}

			totalRevenue := "EUR 0.00"
			if revenue, ok := equipment["totalRevenue"].(float64); ok {
				totalRevenue = fmt.Sprintf("EUR %.2f", revenue)
			}

			pdf.CellFormat(40, 6, deviceID, "1", 0, "L", true, 0, "")
			pdf.CellFormat(60, 6, productName, "1", 0, "L", true, 0, "")
			pdf.CellFormat(30, 6, rentalCount, "1", 0, "C", true, 0, "")
			pdf.CellFormat(35, 6, totalRevenue, "1", 0, "R", true, 0, "")
			pdf.Ln(6)
		}
	}

	pdf.Ln(10)
}

// addTopCustomersTable adds top customers table to PDF
func (h *AnalyticsHandler) addTopCustomersTable(pdf *gofpdf.Fpdf, data interface{}) {
	if pdf.GetY() > 220 {
		pdf.AddPage()
	}

	// Table title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.Cell(190, 10, "Top Customers by Revenue")
	pdf.Ln(12)

	// Table headers
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(70, 8, "Customer Name", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Jobs", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Avg Revenue", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Total Revenue", "1", 0, "C", true, 0, "")
	pdf.Ln(8)

	// Table data
	pdf.SetFont("Arial", "", 8)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetTextColor(51, 51, 51)

	if customerList, ok := data.([]map[string]interface{}); ok {
		for i, customer := range customerList {
			if i >= 10 { // Limit to top 10
				break
			}
			
			// Alternate row colors
			if i%2 == 1 {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			customerName := ""
			if name, ok := customer["customerName"].(string); ok {
				customerName = name
				if len(customerName) > 30 {
					customerName = customerName[:27] + "..."
				}
			}

			jobCount := "0"
			if count, ok := customer["jobCount"].(int); ok {
				jobCount = strconv.Itoa(count)
			}

			avgRevenue := "EUR 0.00"
			if revenue, ok := customer["avgRevenue"].(float64); ok {
				avgRevenue = fmt.Sprintf("EUR %.2f", revenue)
			}

			totalRevenue := "EUR 0.00"
			if revenue, ok := customer["totalRevenue"].(float64); ok {
				totalRevenue = fmt.Sprintf("EUR %.2f", revenue)
			}

			pdf.CellFormat(70, 6, customerName, "1", 0, "L", true, 0, "")
			pdf.CellFormat(30, 6, jobCount, "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 6, avgRevenue, "1", 0, "R", true, 0, "")
			pdf.CellFormat(40, 6, totalRevenue, "1", 0, "R", true, 0, "")
			pdf.Ln(6)
		}
	}

	// Add footer
	pdf.Ln(15)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(190, 6, fmt.Sprintf("Generated on %s by RentalCore Analytics", time.Now().Format("2006-01-02 15:04:05")), "", 0, "C", false, 0, "")
}