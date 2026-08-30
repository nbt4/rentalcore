package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go-barcode-webapp/internal/jobstatus"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dashboardWidgetView struct {
	Key         string
	Title       string
	Description string
	Icon        string
	Link        string
	Value       string
}

type HomeHandler struct {
	jobRepo      *repository.JobRepository
	deviceRepo   *repository.DeviceRepository
	customerRepo *repository.CustomerRepository
	caseRepo     *repository.CaseRepository
	db           *gorm.DB
}

func NewHomeHandler(jobRepo *repository.JobRepository, deviceRepo *repository.DeviceRepository, customerRepo *repository.CustomerRepository, caseRepo *repository.CaseRepository, db *gorm.DB) *HomeHandler {
	return &HomeHandler{
		jobRepo:      jobRepo,
		deviceRepo:   deviceRepo,
		customerRepo: customerRepo,
		caseRepo:     caseRepo,
		db:           db,
	}
}

func (h *HomeHandler) Dashboard(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	storageCoreDomain, rentalCoreDomain := GetAppDomains(c)

	// Get real counts from database using direct queries
	var totalJobs int64
	var activeJobs int64
	var totalDevices int64
	var totalCustomers int64
	var totalCases int64

	// Use the DB connection to count records
	h.db.Model(&models.Job{}).Count(&totalJobs)
	// Planning and confirmed jobs are the only open lifecycle states.
	h.db.Model(&models.Job{}).
		Where("deleted_at IS NULL AND statusid IN ?", jobstatus.OpenIDs).
		Count(&activeJobs)
	h.db.Model(&models.Device{}).Count(&totalDevices)
	h.db.Model(&models.Customer{}).Count(&totalCustomers)
	h.db.Model(&models.Case{}).Count(&totalCases)

	stats := gin.H{
		"TotalJobs":         totalJobs,
		"ActiveJobs":        activeJobs,
		"TotalDevices":      totalDevices,
		"TotalCustomers":    totalCustomers,
		"TotalCases":        totalCases,
		"DefaultWidgetKeys": defaultDashboardWidgetKeys(),
	}

	availableWidgets := getAvailableDashboardWidgets()
	widgetValues := map[string]string{
		"total_jobs":      fmt.Sprintf("%d", totalJobs),
		"active_jobs":     fmt.Sprintf("%d", activeJobs),
		"total_devices":   fmt.Sprintf("%d", totalDevices),
		"total_customers": fmt.Sprintf("%d", totalCustomers),
		"total_cases":     fmt.Sprintf("%d", totalCases),
	}

	activeWidgetKeys := defaultDashboardWidgetKeys()
	if user != nil {
		if storedKeys, err := h.loadUserDashboardWidgets(user.UserID); err == nil && len(storedKeys) > 0 {
			activeWidgetKeys = sanitizeWidgetKeys(storedKeys, availableWidgets)
		}
	}
	if len(activeWidgetKeys) == 0 {
		activeWidgetKeys = defaultDashboardWidgetKeys()
	}

	activeWidgets := make([]dashboardWidgetView, 0, len(activeWidgetKeys))
	for _, widget := range availableWidgets {
		if contains(activeWidgetKeys, widget.Key) {
			activeWidgets = append(activeWidgets, dashboardWidgetView{
				Key:         widget.Key,
				Title:       widget.Title,
				Description: widget.Description,
				Icon:        widget.Icon,
				Link:        widget.Link,
				Value:       widgetValues[widget.Key],
			})
		}
	}

	activeWidgetKeysJSON, err := json.Marshal(activeWidgetKeys)
	if err != nil {
		activeWidgetKeysJSON = []byte("[]")
	}

	activeWidgetSet := make(map[string]bool, len(activeWidgetKeys))
	for _, key := range activeWidgetKeys {
		activeWidgetSet[key] = true
	}

	// Get recent jobs (limit to 5 for performance)
	recentJobs, _ := h.jobRepo.List(&models.FilterParams{
		Limit: 5,
	})

	c.HTML(http.StatusOK, "home.html", gin.H{
		"title":               "Home",
		"user":                user,
		"stats":               stats,
		"recentJobs":          recentJobs,
		"currentPage":         "home",
		"dashboardWidgets":    activeWidgets,
		"availableWidgets":    availableWidgets,
		"activeWidgetKeys":    string(activeWidgetKeysJSON),
		"activeWidgetSet":     activeWidgetSet,
		"defaultWidgetKeys":   defaultDashboardWidgetKeys(),
		"WarehouseCoreDomain": storageCoreDomain,
		"RentalCoreDomain":    rentalCoreDomain,
	})
}

func (h *HomeHandler) loadUserDashboardWidgets(userID uint) ([]string, error) {
	var prefs models.DashboardWidgetPreferences
	if err := h.db.Where("user_id = ?", userID).First(&prefs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if len(prefs.Widgets) == 0 {
		return nil, nil
	}

	var widgetKeys []string
	if err := json.Unmarshal(prefs.Widgets, &widgetKeys); err != nil {
		return nil, err
	}
	return widgetKeys, nil
}

func (h *HomeHandler) saveUserDashboardWidgets(userID uint, widgetKeys []string) error {
	if len(widgetKeys) == 0 {
		return errors.New("no widget keys provided")
	}

	payload, err := json.Marshal(widgetKeys)
	if err != nil {
		return err
	}

	var prefs models.DashboardWidgetPreferences
	err = h.db.Where("user_id = ?", userID).First(&prefs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		prefs = models.DashboardWidgetPreferences{
			UserID:  userID,
			Widgets: payload,
		}
		return h.db.Create(&prefs).Error
	} else if err != nil {
		return err
	}

	prefs.Widgets = payload
	return h.db.Save(&prefs).Error
}

func (h *HomeHandler) GetDashboardWidgetPreferences(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	widgetKeys, err := h.loadUserDashboardWidgets(user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load widget preferences"})
		return
	}

	if len(widgetKeys) == 0 {
		widgetKeys = defaultDashboardWidgetKeys()
	}

	c.JSON(http.StatusOK, gin.H{
		"widgets": widgetKeys,
	})
}

type updateWidgetRequest struct {
	Widgets []string `json:"widgets"`
}

func (h *HomeHandler) UpdateDashboardWidgetPreferences(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateWidgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	if len(req.Widgets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "widgets array cannot be empty"})
		return
	}

	validKeys := sanitizeWidgetKeys(req.Widgets, getAvailableDashboardWidgets())
	if len(validKeys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid widget keys provided"})
		return
	}

	if err := h.saveUserDashboardWidgets(user.UserID, validKeys); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "dashboard widgets updated",
		"widgets": validKeys,
	})
}

func getAvailableDashboardWidgets() []models.DashboardWidgetDefinition {
	return []models.DashboardWidgetDefinition{
		{
			Key:         "total_customers",
			Title:       "Active Customers",
			Description: "Customers currently registered in the system",
			Icon:        "bi-people",
			Link:        "/customers",
		},
		{
			Key:         "total_devices",
			Title:       "Equipment Items",
			Description: "Devices available across the inventory",
			Icon:        "bi-cpu",
			Link:        "/devices",
		},
		{
			Key:         "active_jobs",
			Title:       "Active Jobs",
			Description: "Jobs currently running or pending",
			Icon:        "bi-briefcase",
			Link:        "/jobs",
		},
		{
			Key:         "total_cases",
			Title:       "Equipment Cases",
			Description: "Organized equipment cases across storage",
			Icon:        "bi-box",
			Link:        "/cases",
		},
		{
			Key:         "total_jobs",
			Title:       "Total Jobs",
			Description: "All jobs ever created in RentalCore",
			Icon:        "bi-clipboard-data",
			Link:        "/jobs?scope=all",
		},
	}
}

func defaultDashboardWidgetKeys() []string {
	return []string{"total_customers", "total_devices", "active_jobs", "total_cases"}
}

func sanitizeWidgetKeys(widgetKeys []string, available []models.DashboardWidgetDefinition) []string {
	allowed := make(map[string]struct{}, len(available))
	for _, widget := range available {
		allowed[widget.Key] = struct{}{}
	}

	seen := make(map[string]struct{})
	valid := make([]string, 0, len(widgetKeys))
	for _, key := range widgetKeys {
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		valid = append(valid, key)
	}
	return valid
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
