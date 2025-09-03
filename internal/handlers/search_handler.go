package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

// GlobalSearch performs search across all entities
func (h *SearchHandler) GlobalSearch(c *gin.Context) {
	query := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	searchType := c.DefaultQuery("type", "global")

	// For GET requests without query, always show search page (browser navigation)
	if query == "" && c.Request.Method == "GET" {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusOK, "search_results.html", gin.H{
			"title":      "Global Search",
			"user":       user,
			"query":      "",
			"searchType": searchType,
			"results":    nil,
		})
		return
	}

	// If no query for non-GET requests, return error
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Log search for analytics
	currentUser, _ := GetCurrentUser(c)
	h.logSearch(currentUser.UserID, query, searchType, page, pageSize)

	results := make(map[string]interface{})
	
	if searchType == "global" || searchType == "jobs" {
		results["jobs"] = h.searchJobs(query, page, pageSize)
	}
	
	if searchType == "global" || searchType == "devices" {
		results["devices"] = h.searchDevices(query, page, pageSize)
	}
	
	if searchType == "global" || searchType == "customers" {
		results["customers"] = h.searchCustomers(query, page, pageSize)
	}
	
	if searchType == "global" || searchType == "cases" {
		results["cases"] = h.searchCases(query, page, pageSize)
	}

	// For GET requests, always return HTML (browser navigation)
	// For other methods, return JSON (API calls)
	if c.Request.Method == "GET" {
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusOK, "search_results.html", gin.H{
			"title":      "Search Results",
			"user":       user,
			"query":      query,
			"searchType": searchType,
			"results":    results,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"query":   query,
			"type":    searchType,
			"page":    page,
			"results": results,
		})
	}
}

// searchJobs searches in jobs table
func (h *SearchHandler) searchJobs(query string, page, pageSize int) map[string]interface{} {
	var jobs []models.Job
	var total int64

	offset := (page - 1) * pageSize
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Count total
	h.db.Model(&models.Job{}).
		Where("LOWER(description) LIKE ? OR jobID = ?", searchTerm, query).
		Count(&total)

	// Get results with pagination
	h.db.Preload("Customer").Preload("Status").
		Where("LOWER(description) LIKE ? OR jobID = ?", searchTerm, query).
		Offset(offset).Limit(pageSize).
		Find(&jobs)

	return map[string]interface{}{
		"items": jobs,
		"total": total,
		"page":  page,
	}
}

// searchDevices searches in devices table
func (h *SearchHandler) searchDevices(query string, page, pageSize int) map[string]interface{} {
	var devices []models.Device
	var total int64

	offset := (page - 1) * pageSize
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Count total
	h.db.Model(&models.Device{}).
		Joins("LEFT JOIN products ON devices.productID = products.productID").
		Where("LOWER(devices.deviceID) LIKE ? OR LOWER(devices.serialnumber) LIKE ? OR LOWER(products.name) LIKE ?", 
			searchTerm, searchTerm, searchTerm).
		Count(&total)

	// Get results with pagination
	h.db.Preload("Product").
		Joins("LEFT JOIN products ON devices.productID = products.productID").
		Where("LOWER(devices.deviceID) LIKE ? OR LOWER(devices.serialnumber) LIKE ? OR LOWER(products.name) LIKE ?", 
			searchTerm, searchTerm, searchTerm).
		Offset(offset).Limit(pageSize).
		Find(&devices)

	return map[string]interface{}{
		"items": devices,
		"total": total,
		"page":  page,
	}
}

// searchCustomers searches in customers table
func (h *SearchHandler) searchCustomers(query string, page, pageSize int) map[string]interface{} {
	var customers []models.Customer
	var total int64

	offset := (page - 1) * pageSize
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Count total
	h.db.Model(&models.Customer{}).
		Where("LOWER(companyname) LIKE ? OR LOWER(firstname) LIKE ? OR LOWER(lastname) LIKE ? OR LOWER(email) LIKE ? OR customerID = ?", 
			searchTerm, searchTerm, searchTerm, searchTerm, query).
		Count(&total)

	// Get results with pagination
	h.db.Where("LOWER(companyname) LIKE ? OR LOWER(firstname) LIKE ? OR LOWER(lastname) LIKE ? OR LOWER(email) LIKE ? OR customerID = ?", 
		searchTerm, searchTerm, searchTerm, searchTerm, query).
		Offset(offset).Limit(pageSize).
		Find(&customers)

	return map[string]interface{}{
		"items": customers,
		"total": total,
		"page":  page,
	}
}

// searchCases searches in cases table
func (h *SearchHandler) searchCases(query string, page, pageSize int) map[string]interface{} {
	var cases []models.Case
	var total int64

	offset := (page - 1) * pageSize
	searchTerm := "%" + strings.ToLower(query) + "%"

	// Count total
	h.db.Model(&models.Case{}).
		Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR caseID = ?", 
			searchTerm, searchTerm, query).
		Count(&total)

	// Get results with pagination
	h.db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR caseID = ?", 
		searchTerm, searchTerm, query).
		Offset(offset).Limit(pageSize).
		Find(&cases)

	return map[string]interface{}{
		"items": cases,
		"total": total,
		"page":  page,
	}
}

// AdvancedSearch handles complex filtering and search
func (h *SearchHandler) AdvancedSearch(c *gin.Context) {
	var request struct {
		Query      string                 `json:"query"`
		Type       string                 `json:"type"`
		Filters    map[string]interface{} `json:"filters"`
		Sort       string                 `json:"sort"`
		Page       int                    `json:"page"`
		PageSize   int                    `json:"pageSize"`
		SaveSearch bool                   `json:"saveSearch"`
		SearchName string                 `json:"searchName"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if request.Page == 0 {
		request.Page = 1
	}
	if request.PageSize == 0 {
		request.PageSize = 20
	}

	var results interface{}
	var total int64

	switch request.Type {
	case "jobs":
		results, total = h.advancedSearchJobs(request.Query, request.Filters, request.Sort, request.Page, request.PageSize)
	case "devices":
		results, total = h.advancedSearchDevices(request.Query, request.Filters, request.Sort, request.Page, request.PageSize)
	case "customers":
		results, total = h.advancedSearchCustomers(request.Query, request.Filters, request.Sort, request.Page, request.PageSize)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search type"})
		return
	}

	// Save search if requested
	if request.SaveSearch && request.SearchName != "" {
		currentUser, _ := GetCurrentUser(c)
		h.saveSearch(currentUser.UserID, request.SearchName, request.Type, request.Filters)
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   total,
		"page":    request.Page,
		"pageSize": request.PageSize,
	})
}

// advancedSearchJobs performs advanced job search with filters
func (h *SearchHandler) advancedSearchJobs(query string, filters map[string]interface{}, sort string, page, pageSize int) ([]models.Job, int64) {
	var jobs []models.Job
	var total int64

	db := h.db.Model(&models.Job{}).Preload("Customer").Preload("Status")

	// Apply text search
	if query != "" {
		searchTerm := "%" + strings.ToLower(query) + "%"
		db = db.Where("LOWER(description) LIKE ? OR jobID = ?", searchTerm, query)
	}

	// Apply filters
	if customerID, ok := filters["customerID"]; ok && customerID != "" {
		db = db.Where("customerID = ?", customerID)
	}
	
	if statusID, ok := filters["statusID"]; ok && statusID != "" {
		db = db.Where("statusID = ?", statusID)
	}
	
	if startDate, ok := filters["startDate"]; ok && startDate != "" {
		db = db.Where("startDate >= ?", startDate)
	}
	
	if endDate, ok := filters["endDate"]; ok && endDate != "" {
		db = db.Where("endDate <= ?", endDate)
	}
	
	if minRevenue, ok := filters["minRevenue"]; ok && minRevenue != "" {
		db = db.Where("final_revenue >= ?", minRevenue)
	}
	
	if maxRevenue, ok := filters["maxRevenue"]; ok && maxRevenue != "" {
		db = db.Where("final_revenue <= ?", maxRevenue)
	}

	// Count total
	db.Count(&total)

	// Apply sorting
	switch sort {
	case "revenue_desc":
		db = db.Order("final_revenue DESC")
	case "revenue_asc":
		db = db.Order("final_revenue ASC")
	case "date_desc":
		db = db.Order("startDate DESC")
	case "date_asc":
		db = db.Order("startDate ASC")
	default:
		db = db.Order("jobID DESC")
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	db.Offset(offset).Limit(pageSize).Find(&jobs)

	return jobs, total
}

// advancedSearchDevices performs advanced device search with filters
func (h *SearchHandler) advancedSearchDevices(query string, filters map[string]interface{}, sort string, page, pageSize int) ([]models.Device, int64) {
	var devices []models.Device
	var total int64

	db := h.db.Model(&models.Device{}).Preload("Product")

	// Apply text search
	if query != "" {
		searchTerm := "%" + strings.ToLower(query) + "%"
		db = db.Joins("LEFT JOIN products ON devices.productID = products.productID").
			Where("LOWER(devices.deviceID) LIKE ? OR LOWER(devices.serialnumber) LIKE ? OR LOWER(products.name) LIKE ?", 
				searchTerm, searchTerm, searchTerm)
	}

	// Apply filters
	if status, ok := filters["status"]; ok && status != "" {
		db = db.Where("status = ?", status)
	}
	
	if productID, ok := filters["productID"]; ok && productID != "" {
		db = db.Where("devices.productID = ?", productID)
	}
	
	if purchaseDateFrom, ok := filters["purchaseDateFrom"]; ok && purchaseDateFrom != "" {
		db = db.Where("purchaseDate >= ?", purchaseDateFrom)
	}
	
	if purchaseDateTo, ok := filters["purchaseDateTo"]; ok && purchaseDateTo != "" {
		db = db.Where("purchaseDate <= ?", purchaseDateTo)
	}

	// Count total
	db.Count(&total)

	// Apply sorting
	switch sort {
	case "device_id":
		db = db.Order("deviceID ASC")
	case "product_name":
		db = db.Joins("LEFT JOIN products ON devices.productID = products.productID").Order("products.name ASC")
	case "purchase_date_desc":
		db = db.Order("purchaseDate DESC")
	case "purchase_date_asc":
		db = db.Order("purchaseDate ASC")
	default:
		db = db.Order("deviceID ASC")
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	db.Offset(offset).Limit(pageSize).Find(&devices)

	return devices, total
}

// advancedSearchCustomers performs advanced customer search with filters
func (h *SearchHandler) advancedSearchCustomers(query string, filters map[string]interface{}, sort string, page, pageSize int) ([]models.Customer, int64) {
	var customers []models.Customer
	var total int64

	db := h.db.Model(&models.Customer{})

	// Apply text search
	if query != "" {
		searchTerm := "%" + strings.ToLower(query) + "%"
		db = db.Where("LOWER(companyname) LIKE ? OR LOWER(firstname) LIKE ? OR LOWER(lastname) LIKE ? OR LOWER(email) LIKE ?", 
			searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// Apply filters
	if customerType, ok := filters["customertype"]; ok && customerType != "" {
		db = db.Where("customertype = ?", customerType)
	}
	
	if city, ok := filters["city"]; ok && city != "" {
		db = db.Where("city = ?", city)
	}
	
	if country, ok := filters["country"]; ok && country != "" {
		db = db.Where("country = ?", country)
	}

	// Count total
	db.Count(&total)

	// Apply sorting
	switch sort {
	case "company_name":
		db = db.Order("companyname ASC")
	case "last_name":
		db = db.Order("lastname ASC")
	case "created_desc":
		db = db.Order("created_at DESC")
	case "created_asc":
		db = db.Order("created_at ASC")
	default:
		db = db.Order("customerID DESC")
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	db.Offset(offset).Limit(pageSize).Find(&customers)

	return customers, total
}

// SavedSearches returns user's saved searches
func (h *SearchHandler) SavedSearches(c *gin.Context) {
	currentUser, _ := GetCurrentUser(c)
	searchType := c.Query("type")

	var savedSearches []models.SavedSearch
	query := h.db.Where("userID = ?", currentUser.UserID)
	
	if searchType != "" {
		query = query.Where("search_type = ?", searchType)
	}
	
	query.Order("usage_count DESC, updated_at DESC").Find(&savedSearches)

	c.JSON(http.StatusOK, gin.H{"savedSearches": savedSearches})
}

// DeleteSavedSearch deletes a saved search
func (h *SearchHandler) DeleteSavedSearch(c *gin.Context) {
	searchID := c.Param("id")
	currentUser, _ := GetCurrentUser(c)

	result := h.db.Where("searchID = ? AND userID = ?", searchID, currentUser.UserID).
		Delete(&models.SavedSearch{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Saved search not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Saved search deleted"})
}

// SearchSuggestions provides autocomplete suggestions
func (h *SearchHandler) SearchSuggestions(c *gin.Context) {
	query := c.Query("q")
	searchType := c.Query("type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if query == "" {
		c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
		return
	}

	suggestions := make([]string, 0)
	searchTerm := "%" + strings.ToLower(query) + "%"

	switch searchType {
	case "customers":
		var names []string
		h.db.Model(&models.Customer{}).
			Select("DISTINCT COALESCE(companyname, CONCAT(firstname, ' ', lastname)) as name").
			Where("LOWER(COALESCE(companyname, CONCAT(firstname, ' ', lastname))) LIKE ?", searchTerm).
			Limit(limit).
			Pluck("name", &names)
		suggestions = append(suggestions, names...)

	case "devices":
		var deviceIDs []string
		h.db.Model(&models.Device{}).
			Select("DISTINCT deviceID").
			Where("LOWER(deviceID) LIKE ?", searchTerm).
			Limit(limit).
			Pluck("deviceID", &deviceIDs)
		suggestions = append(suggestions, deviceIDs...)

	case "jobs":
		var descriptions []string
		h.db.Model(&models.Job{}).
			Select("DISTINCT description").
			Where("LOWER(description) LIKE ?", searchTerm).
			Limit(limit).
			Pluck("description", &descriptions)
		suggestions = append(suggestions, descriptions...)
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// logSearch logs search for analytics
func (h *SearchHandler) logSearch(userID uint, query, searchType string, page, pageSize int) {
	searchHistory := models.SearchHistory{
		UserID:     &userID,
		SearchTerm: query,
		SearchType: searchType,
		SearchedAt: time.Now(),
	}
	h.db.Create(&searchHistory)
}

// saveSearch saves a search for later use
func (h *SearchHandler) saveSearch(userID uint, name, searchType string, filters map[string]interface{}) {
	// Convert filters to JSON
	filtersJSON, _ := json.Marshal(filters)

	savedSearch := models.SavedSearch{
		UserID:     userID,
		Name:       name,
		SearchType: searchType,
		Filters:    filtersJSON,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	h.db.Create(&savedSearch)
}