package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RevenueDrilldownNode struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Label         string                 `json:"label"`
	Revenue       float64                `json:"revenue"`
	Cost          float64                `json:"cost"`
	Margin        float64                `json:"margin"`
	MarginPercent float64                `json:"margin_percent"`
	HasCost       bool                   `json:"has_cost"`
	Quantity      float64                `json:"quantity"`
	Bookings      int                    `json:"bookings"`
	Children      []RevenueDrilldownNode `json:"children"`
}

type RevenueDrilldownResponse struct {
	Period              string                 `json:"period"`
	StartDate           string                 `json:"start_date,omitempty"`
	EndDate             string                 `json:"end_date,omitempty"`
	TotalRevenue        float64                `json:"total_revenue"`
	AttributedRevenue   float64                `json:"attributed_revenue"`
	UnattributedRevenue float64                `json:"unattributed_revenue"`
	RentalRevenue       float64                `json:"rental_revenue"`
	RentalCost          float64                `json:"rental_cost"`
	RentalMargin        float64                `json:"rental_margin"`
	RentalMarginPercent float64                `json:"rental_margin_percent"`
	JobCount            int                    `json:"job_count"`
	Categories          []RevenueDrilldownNode `json:"categories"`
}

type revenueDrilldownJob struct {
	JobID          uint       `gorm:"column:job_id"`
	Revenue        float64    `gorm:"column:revenue"`
	StartDate      *time.Time `gorm:"column:start_date"`
	EndDate        *time.Time `gorm:"column:end_date"`
	MultiplyByDays bool       `gorm:"column:multiply_by_days"`
}

type revenueDrilldownPosition struct {
	PositionID        uint    `gorm:"column:position_id"`
	JobID             uint    `gorm:"column:job_id"`
	PositionType      string  `gorm:"column:position_type"`
	ProductID         *uint   `gorm:"column:product_id"`
	ServiceItemID     *uint   `gorm:"column:service_item_id"`
	RentalEquipmentID *uint   `gorm:"column:rental_equipment_id"`
	Description       string  `gorm:"column:description"`
	ItemName          string  `gorm:"column:item_name"`
	Quantity          float64 `gorm:"column:quantity"`
	UnitPrice         float64 `gorm:"column:unit_price"`
	FollowDayFactor   float64 `gorm:"column:follow_day_factor"`
	DiscountPercent   float64 `gorm:"column:discount_percent"`
	DiscountAmount    float64 `gorm:"column:discount_amount"`
	SupplierUnitCost  float64 `gorm:"column:supplier_unit_cost"`
}

type revenueDrilldownDevice struct {
	PositionID   uint   `gorm:"column:position_id"`
	DeviceID     string `gorm:"column:device_id"`
	SerialNumber string `gorm:"column:serial_number"`
}

type revenueDrilldownRentalCost struct {
	JobID       uint    `gorm:"column:job_id"`
	EquipmentID uint    `gorm:"column:equipment_id"`
	ItemName    string  `gorm:"column:item_name"`
	TotalCost   float64 `gorm:"column:total_cost"`
}

type revenueAggregationNode struct {
	data     RevenueDrilldownNode
	children map[string]*revenueAggregationNode
	jobs     map[uint]struct{}
}

func newRevenueAggregationNode(id, nodeType, label string) *revenueAggregationNode {
	return &revenueAggregationNode{
		data:     RevenueDrilldownNode{ID: id, Type: nodeType, Label: label, Children: []RevenueDrilldownNode{}},
		children: make(map[string]*revenueAggregationNode),
		jobs:     make(map[uint]struct{}),
	}
}

func (n *revenueAggregationNode) add(revenue, cost, quantity float64, hasCost bool, jobID uint) {
	n.data.Revenue += revenue
	n.data.Cost += cost
	n.data.Quantity += quantity
	n.data.HasCost = n.data.HasCost || hasCost
	if jobID != 0 {
		n.jobs[jobID] = struct{}{}
	}
}

func (n *revenueAggregationNode) child(id, nodeType, label string) *revenueAggregationNode {
	if existing, ok := n.children[id]; ok {
		return existing
	}
	child := newRevenueAggregationNode(id, nodeType, label)
	n.children[id] = child
	return child
}

func (n *revenueAggregationNode) finalize() RevenueDrilldownNode {
	n.data.Revenue = roundAnalyticsMoney(n.data.Revenue)
	n.data.Cost = roundAnalyticsMoney(n.data.Cost)
	n.data.Bookings = len(n.jobs)
	if n.data.HasCost {
		n.data.Margin = roundAnalyticsMoney(n.data.Revenue - n.data.Cost)
		if n.data.Revenue != 0 {
			n.data.MarginPercent = math.Round((n.data.Margin/n.data.Revenue)*1000) / 10
		}
	}

	n.data.Children = make([]RevenueDrilldownNode, 0, len(n.children))
	for _, child := range n.children {
		n.data.Children = append(n.data.Children, child.finalize())
	}
	sort.SliceStable(n.data.Children, func(i, j int) bool {
		if n.data.Children[i].Revenue == n.data.Children[j].Revenue {
			return n.data.Children[i].Label < n.data.Children[j].Label
		}
		return n.data.Children[i].Revenue > n.data.Children[j].Revenue
	})
	return n.data
}

func roundAnalyticsMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func analyticsEventDays(job revenueDrilldownJob) int {
	if job.StartDate == nil || job.EndDate == nil {
		return 1
	}
	days := int(job.EndDate.Sub(*job.StartDate).Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

func analyticsLineRevenue(job revenueDrilldownJob, position revenueDrilldownPosition) float64 {
	dayFactor := 1.0
	if job.MultiplyByDays && analyticsEventDays(job) > 1 && position.FollowDayFactor > 0 {
		dayFactor = 1 + float64(analyticsEventDays(job)-1)*position.FollowDayFactor
	}
	lineTotal := position.Quantity * position.UnitPrice * dayFactor
	discount := position.DiscountAmount + lineTotal*position.DiscountPercent/100
	return math.Max(0, lineTotal-discount)
}

func analyticsRentalCost(job revenueDrilldownJob, position revenueDrilldownPosition) float64 {
	dayFactor := 1.0
	if job.MultiplyByDays {
		dayFactor = float64(analyticsEventDays(job))
	}
	return position.SupplierUnitCost * math.Max(position.Quantity, 1) * dayFactor
}

func drilldownItemKey(prefix string, id *uint, label string) string {
	if id != nil {
		return fmt.Sprintf("%s:%d", prefix, *id)
	}
	return prefix + ":" + strings.ToLower(strings.TrimSpace(label))
}

func drilldownItemLabel(position revenueDrilldownPosition, fallback string) string {
	if strings.TrimSpace(position.ItemName) != "" {
		return position.ItemName
	}
	if strings.TrimSpace(position.Description) != "" {
		return position.Description
	}
	return fallback
}

func buildRevenueDrilldown(
	period string,
	startDate, endDate *time.Time,
	jobs []revenueDrilldownJob,
	positions []revenueDrilldownPosition,
	devices []revenueDrilldownDevice,
	rentalCosts []revenueDrilldownRentalCost,
) RevenueDrilldownResponse {
	categories := []*revenueAggregationNode{
		newRevenueAggregationNode("own-products", "category", "Eigene Produkte"),
		newRevenueAggregationNode("rental-products", "category", "Mietprodukte"),
		newRevenueAggregationNode("services", "category", "Dienstleistungen"),
		newRevenueAggregationNode("packages", "category", "Pakete"),
		newRevenueAggregationNode("unattributed", "category", "Nicht zugeordneter Umsatz"),
	}
	categoryByType := map[string]*revenueAggregationNode{
		"product": categories[0],
		"rental":  categories[1],
		"service": categories[2],
		"package": categories[3],
	}

	jobsByID := make(map[uint]revenueDrilldownJob, len(jobs))
	positionsByJob := make(map[uint][]revenueDrilldownPosition)
	devicesByPosition := make(map[uint][]revenueDrilldownDevice)
	for _, job := range jobs {
		jobsByID[job.JobID] = job
	}
	for _, position := range positions {
		positionsByJob[position.JobID] = append(positionsByJob[position.JobID], position)
	}
	for _, device := range devices {
		devicesByPosition[device.PositionID] = append(devicesByPosition[device.PositionID], device)
	}

	rentalCostsByKey := make(map[string]revenueDrilldownRentalCost, len(rentalCosts))
	for _, cost := range rentalCosts {
		key := fmt.Sprintf("%d:%d", cost.JobID, cost.EquipmentID)
		rentalCostsByKey[key] = cost
	}
	rentalPositionsByKey := make(map[string][]revenueDrilldownPosition)
	for _, position := range positions {
		if position.PositionType == "rental" && position.RentalEquipmentID != nil {
			key := fmt.Sprintf("%d:%d", position.JobID, *position.RentalEquipmentID)
			rentalPositionsByKey[key] = append(rentalPositionsByKey[key], position)
		}
	}
	rentalCostByPosition := make(map[uint]float64)
	for key, groupedPositions := range rentalPositionsByKey {
		storedCost, hasStoredCost := rentalCostsByKey[key]
		totalWeight := 0.0
		weights := make(map[uint]float64, len(groupedPositions))
		for _, position := range groupedPositions {
			job := jobsByID[position.JobID]
			weight := analyticsRentalCost(job, position)
			if weight <= 0 {
				weight = math.Max(position.Quantity, 1)
			}
			weights[position.PositionID] = weight
			totalWeight += weight
		}
		for _, position := range groupedPositions {
			if hasStoredCost && totalWeight > 0 {
				rentalCostByPosition[position.PositionID] = storedCost.TotalCost * weights[position.PositionID] / totalWeight
			} else {
				rentalCostByPosition[position.PositionID] = analyticsRentalCost(jobsByID[position.JobID], position)
			}
		}
	}

	totalRevenue := 0.0
	attributedRevenue := 0.0
	for _, job := range jobs {
		totalRevenue += job.Revenue
		jobPositions := positionsByJob[job.JobID]
		rawTotal := 0.0
		rawByPosition := make(map[uint]float64, len(jobPositions))
		for _, position := range jobPositions {
			raw := analyticsLineRevenue(job, position)
			rawByPosition[position.PositionID] = raw
			rawTotal += raw
		}
		if rawTotal <= 0 {
			categories[4].add(job.Revenue, 0, 0, false, job.JobID)
			continue
		}

		for _, position := range jobPositions {
			category, ok := categoryByType[position.PositionType]
			if !ok {
				category = categories[4]
			}
			allocatedRevenue := job.Revenue * rawByPosition[position.PositionID] / rawTotal
			attributedRevenue += allocatedRevenue
			cost := rentalCostByPosition[position.PositionID]
			hasCost := position.PositionType == "rental"
			category.add(allocatedRevenue, cost, position.Quantity, hasCost, job.JobID)

			var itemID, itemType, itemLabel string
			switch position.PositionType {
			case "product":
				itemLabel = drilldownItemLabel(position, "Unbekanntes Produkt")
				itemID = drilldownItemKey("product", position.ProductID, itemLabel)
				itemType = "product"
			case "rental":
				itemLabel = drilldownItemLabel(position, "Unbekanntes Mietprodukt")
				itemID = drilldownItemKey("rental", position.RentalEquipmentID, itemLabel)
				itemType = "rental_product"
			case "service":
				itemLabel = drilldownItemLabel(position, "Unbekannte Dienstleistung")
				itemID = drilldownItemKey("service", position.ServiceItemID, itemLabel)
				itemType = "service"
			case "package":
				itemLabel = drilldownItemLabel(position, "Unbekanntes Paket")
				itemID = drilldownItemKey("package", position.ProductID, itemLabel)
				itemType = "package"
			default:
				continue
			}

			item := category.child(itemID, itemType, itemLabel)
			item.add(allocatedRevenue, cost, position.Quantity, hasCost, job.JobID)
			if position.PositionType != "product" {
				continue
			}

			positionDevices := devicesByPosition[position.PositionID]
			allocationUnits := math.Max(position.Quantity, float64(len(positionDevices)))
			allocationUnits = math.Max(allocationUnits, 1)
			deviceRevenue := allocatedRevenue / allocationUnits
			for _, device := range positionDevices {
				label := device.DeviceID
				if device.SerialNumber != "" && device.SerialNumber != device.DeviceID {
					label += " · S/N " + device.SerialNumber
				}
				deviceNode := item.child("device:"+device.DeviceID, "device", label)
				deviceNode.add(deviceRevenue, 0, 1, false, job.JobID)
			}
			unassignedUnits := allocationUnits - float64(len(positionDevices))
			if unassignedUnits > 0 {
				unassigned := item.child(itemID+":unassigned", "device", "Noch keinem Gerät zugeordnet")
				unassigned.add(deviceRevenue*unassignedUnits, 0, unassignedUnits, false, job.JobID)
			}
		}
	}

	for key, storedCost := range rentalCostsByKey {
		if _, matched := rentalPositionsByKey[key]; matched {
			continue
		}
		label := storedCost.ItemName
		if strings.TrimSpace(label) == "" {
			label = fmt.Sprintf("Mietprodukt #%d", storedCost.EquipmentID)
		}
		category := categories[1]
		category.add(0, storedCost.TotalCost, 0, true, storedCost.JobID)
		item := category.child(fmt.Sprintf("rental:%d", storedCost.EquipmentID), "rental_product", label)
		item.add(0, storedCost.TotalCost, 0, true, storedCost.JobID)
	}

	response := RevenueDrilldownResponse{
		Period:              period,
		TotalRevenue:        roundAnalyticsMoney(totalRevenue),
		AttributedRevenue:   roundAnalyticsMoney(attributedRevenue),
		UnattributedRevenue: roundAnalyticsMoney(categories[4].data.Revenue),
		JobCount:            len(jobs),
		Categories:          make([]RevenueDrilldownNode, 0, len(categories)),
	}
	if startDate != nil {
		response.StartDate = startDate.Format("2006-01-02")
	}
	if endDate != nil {
		response.EndDate = endDate.Format("2006-01-02")
	}
	for _, category := range categories {
		response.Categories = append(response.Categories, category.finalize())
	}
	rental := response.Categories[1]
	response.RentalRevenue = rental.Revenue
	response.RentalCost = rental.Cost
	response.RentalMargin = rental.Margin
	response.RentalMarginPercent = rental.MarginPercent
	return response
}

func revenueDrilldownPeriod(period string, now time.Time) (*time.Time, *time.Time, error) {
	end := now
	var start time.Time
	switch period {
	case "30days":
		start = end.AddDate(0, 0, -30)
	case "90days":
		start = end.AddDate(0, 0, -90)
	case "1year":
		start = end.AddDate(-1, 0, 0)
	case "all":
		return nil, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported period %q", period)
	}
	return &start, &end, nil
}

// GetRevenueDrilldown returns the reconciled revenue hierarchy used by the analysis page.
func (h *AnalyticsHandler) GetRevenueDrilldown(c *gin.Context) {
	period := c.DefaultQuery("period", "all")
	startDate, endDate, err := revenueDrilldownPeriod(period, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ungültiger Analysezeitraum"})
		return
	}

	dateFilter := ""
	args := []interface{}{}
	if startDate != nil && endDate != nil {
		dateFilter = " AND COALESCE(j.enddate, j.startdate, j.created_at::date) BETWEEN ? AND ?"
		args = append(args, *startDate, *endDate)
	}

	var jobs []revenueDrilldownJob
	if err := h.db.Raw(`
		SELECT j.jobid AS job_id,
		       COALESCE(j.final_revenue, j.revenue, 0) AS revenue,
		       j.startdate AS start_date,
		       j.enddate AS end_date,
		       j.multiply_by_days
		FROM jobs j
		WHERE j.deleted_at IS NULL`+dateFilter+`
		ORDER BY j.jobid`, args...).Scan(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Umsatzdaten konnten nicht geladen werden"})
		return
	}

	var positions []revenueDrilldownPosition
	if err := h.db.Raw(`
		SELECT jp.position_id, jp.job_id, jp.position_type,
		       jp.product_id, jp.service_item_id, jp.rental_equipment_id,
		       jp.description,
		       COALESCE(p.name, s.name, r.name, jp.description, '') AS item_name,
		       jp.quantity, jp.unit_price, jp.follow_day_factor,
		       jp.discount_percent, jp.discount_amount,
		       COALESCE(r.rental_price, 0) AS supplier_unit_cost
		FROM job_positions jp
		JOIN jobs j ON j.jobid = jp.job_id
		LEFT JOIN products p ON p.productid = jp.product_id
		LEFT JOIN service_items s ON s.id = jp.service_item_id
		LEFT JOIN rental_equipment r ON r.id = jp.rental_equipment_id
		WHERE j.deleted_at IS NULL`+dateFilter+`
		ORDER BY jp.job_id, jp.sort_order, jp.position_id`, args...).Scan(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Umsatzpositionen konnten nicht geladen werden"})
		return
	}

	var devices []revenueDrilldownDevice
	if err := h.db.Raw(`
		SELECT jpd.position_id, jpd.device_id,
		       COALESCE(d.serialnumber, '') AS serial_number
		FROM job_position_devices jpd
		JOIN job_positions jp ON jp.position_id = jpd.position_id
		JOIN jobs j ON j.jobid = jp.job_id
		LEFT JOIN devices d ON d.deviceid = jpd.device_id
		WHERE j.deleted_at IS NULL`+dateFilter+`
		ORDER BY jpd.position_id, jpd.device_id`, args...).Scan(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gerätezuordnungen konnten nicht geladen werden"})
		return
	}

	var rentalCosts []revenueDrilldownRentalCost
	if err := h.db.Raw(`
		SELECT jre.job_id, jre.equipment_id,
		       COALESCE(r.name, '') AS item_name,
		       jre.total_cost
		FROM job_rental_equipment jre
		JOIN jobs j ON j.jobid = jre.job_id
		LEFT JOIN rental_equipment r ON r.id = jre.equipment_id
		WHERE j.deleted_at IS NULL`+dateFilter+`
		ORDER BY jre.job_id, jre.equipment_id`, args...).Scan(&rentalCosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mietkosten konnten nicht geladen werden"})
		return
	}

	response := buildRevenueDrilldown(period, startDate, endDate, jobs, positions, devices, rentalCosts)
	c.JSON(http.StatusOK, response)
}
