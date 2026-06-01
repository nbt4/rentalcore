package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/services"
	"go-barcode-webapp/internal/services/warehousecore"

	"github.com/gin-gonic/gin"
)

const (
	jobDebugLogsEnabled  = false
	jobEditingSessionTTL = 2 * time.Minute
)

func jobDebugLog(format string, args ...interface{}) {
	if !jobDebugLogsEnabled {
		return
	}
	fmt.Printf(format, args...)
}

func normalizeProductSelections(selections []JobProductSelection) []JobProductSelection {
	aggregated := make(map[uint]int)
	for _, selection := range selections {
		if selection.ProductID == 0 {
			continue
		}
		if selection.Quantity <= 0 {
			continue
		}
		aggregated[selection.ProductID] += selection.Quantity
	}

	if len(aggregated) == 0 {
		return nil
	}

	result := make([]JobProductSelection, 0, len(aggregated))
	for productID, qty := range aggregated {
		result = append(result, JobProductSelection{ProductID: productID, Quantity: qty})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ProductID < result[j].ProductID
	})

	return result
}

func parseProductSelectionsFromString(raw string) ([]JobProductSelection, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}

	var selections []JobProductSelection
	if err := json.Unmarshal([]byte(clean), &selections); err != nil {
		return nil, err
	}

	return normalizeProductSelections(selections), nil
}

func parseProductSelectionsFromInterface(value interface{}) ([]JobProductSelection, error) {
	switch v := value.(type) {
	case string:
		return parseProductSelectionsFromString(v)
	case []interface{}:
		selections := make([]JobProductSelection, 0, len(v))
		for _, item := range v {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			selection := JobProductSelection{}
			if productIDVal, exists := obj["product_id"]; exists {
				switch id := productIDVal.(type) {
				case float64:
					selection.ProductID = uint(id)
				case int:
					selection.ProductID = uint(id)
				case uint:
					selection.ProductID = id
				}
			}
			if quantityVal, exists := obj["quantity"]; exists {
				switch qty := quantityVal.(type) {
				case float64:
					selection.Quantity = int(qty)
				case int:
					selection.Quantity = qty
				}
			}
			selections = append(selections, selection)
		}
		return normalizeProductSelections(selections), nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported selected_products payload")
	}
}

// CalendarSyncServiceInterface ermöglicht nil-sicheres Einbinden ohne M365-Pflicht.
type CalendarSyncServiceInterface interface {
	SyncJobEvent(jobID uint)
	DeleteJobEvent(jobID uint)
	SyncEmployeeEvent(jobID, employeeID uint)
	DeleteEmployeeEvent(jobID, employeeID uint)
}

type JobHandler struct {
	jobRepo            *repository.JobRepository
	jobPackageRepo     *repository.JobPackageRepository
	deviceRepo         *repository.DeviceRepository
	requirementRepo    *repository.RequirementRepository
	customerRepo       *repository.CustomerRepository
	statusRepo         *repository.StatusRepository
	jobCategoryRepo    *repository.JobCategoryRepository
	jobEditSessionRepo *repository.JobEditSessionRepository
	jobHistoryService  *services.JobHistoryService
	rentalEquipRepo    *repository.RentalEquipmentRepository
	warehouseClient    *warehousecore.Client
	jobEmployeeRepo    *repository.JobEmployeeRepository
	calendarSync       CalendarSyncServiceInterface
}

type JobProductSelection struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

type RentalEquipmentSelection struct {
	EquipmentID uint   `json:"equipment_id"`
	Quantity    uint   `json:"quantity"`
	DaysUsed    uint   `json:"days_used"`
	Notes       string `json:"notes"`
}

func parseRentalEquipmentSelections(raw string) ([]RentalEquipmentSelection, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return nil, nil
	}

	var selections []RentalEquipmentSelection
	if err := json.Unmarshal([]byte(clean), &selections); err != nil {
		return nil, err
	}

	return selections, nil
}

func formatUserDisplayName(user *models.User) string {
	if user == nil {
		return ""
	}

	first := strings.TrimSpace(user.FirstName)
	last := strings.TrimSpace(user.LastName)

	switch {
	case first != "" && last != "":
		return fmt.Sprintf("%s %s", first, last)
	case first != "":
		return first
	case last != "":
		return last
	default:
		return user.Username
	}
}

func NewJobHandler(jobRepo *repository.JobRepository, jobPackageRepo *repository.JobPackageRepository, deviceRepo *repository.DeviceRepository, requirementRepo *repository.RequirementRepository, customerRepo *repository.CustomerRepository, statusRepo *repository.StatusRepository, jobCategoryRepo *repository.JobCategoryRepository, jobEditSessionRepo *repository.JobEditSessionRepository, jobHistoryService *services.JobHistoryService, rentalEquipRepo *repository.RentalEquipmentRepository, jobEmployeeRepo *repository.JobEmployeeRepository, calendarSync CalendarSyncServiceInterface) *JobHandler {
	return &JobHandler{
		jobRepo:            jobRepo,
		jobPackageRepo:     jobPackageRepo,
		deviceRepo:         deviceRepo,
		requirementRepo:    requirementRepo,
		customerRepo:       customerRepo,
		statusRepo:         statusRepo,
		jobCategoryRepo:    jobCategoryRepo,
		jobEditSessionRepo: jobEditSessionRepo,
		jobHistoryService:  jobHistoryService,
		rentalEquipRepo:    rentalEquipRepo,
		warehouseClient:    warehousecore.NewClient(),
		jobEmployeeRepo:    jobEmployeeRepo,
		calendarSync:       calendarSync,
	}
}

// processRentalEquipmentSelections handles adding/updating rental equipment to a job
func (h *JobHandler) processRentalEquipmentSelections(jobID uint, selections []RentalEquipmentSelection) error {
	if h.rentalEquipRepo == nil {
		return nil
	}

	// First, remove all existing rental equipment for this job
	var existingRentals []models.JobRentalEquipment
	if err := h.rentalEquipRepo.GetJobRentalEquipment(jobID, &existingRentals); err == nil {
		for _, rental := range existingRentals {
			h.rentalEquipRepo.RemoveRentalFromJob(jobID, rental.EquipmentID)
		}
	}

	// Add the new selections
	for _, selection := range selections {
		if selection.Quantity == 0 {
			continue
		}

		daysUsed := selection.DaysUsed
		if daysUsed == 0 {
			daysUsed = 1
		}

		jobRental := &models.JobRentalEquipment{
			JobID:       jobID,
			EquipmentID: selection.EquipmentID,
			Quantity:    selection.Quantity,
			DaysUsed:    daysUsed,
			Notes:       selection.Notes,
		}

		if err := h.rentalEquipRepo.AddRentalToJob(jobRental); err != nil {
			return fmt.Errorf("failed to add rental equipment %d: %v", selection.EquipmentID, err)
		}
	}

	return nil
}

func (h *JobHandler) renderJobFormWithError(c *gin.Context, job *models.Job, title, errorText string) {
	user, _ := GetCurrentUser(c)
	customers, _ := h.customerRepo.List(&models.FilterParams{})
	statuses, _ := h.statusRepo.List()
	jobCategories, _ := h.jobCategoryRepo.List()

	// Fetch rental equipment from WarehouseCore
	rentalEquipBySupplier, _ := h.warehouseClient.GetRentalEquipmentBySupplier()

	// Get existing job rental equipment if editing
	var jobRentalEquipment []models.JobRentalEquipment
	if job != nil && job.JobID > 0 {
		h.rentalEquipRepo.GetJobRentalEquipment(job.JobID, &jobRentalEquipment)
	}

	c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
		"title":                 title,
		"job":                   job,
		"customers":             customers,
		"statuses":              statuses,
		"jobCategories":         jobCategories,
		"error":                 errorText,
		"user":                  user,
		"rentalEquipBySupplier": rentalEquipBySupplier,
		"jobRentalEquipment":    jobRentalEquipment,
	})
}

// Web interface handlers
func (h *JobHandler) ListJobs(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// DEBUG: Log all query parameters
	jobDebugLog("DEBUG Job Handler: All query params: %+v\n", c.Request.URL.Query())

	// Manual parameter extraction to ensure search works
	searchParam := c.Query("search")
	jobDebugLog("DEBUG Job Handler: Raw search parameter: '%s'\n", searchParam)
	if searchParam != "" {
		params.SearchTerm = searchParam
		jobDebugLog("DEBUG Job Handler: Search parameter SET to: '%s'\n", searchParam)
	}

	// DEBUG: Log params after binding
	jobDebugLog("DEBUG Job Handler: Final params: SearchTerm='%s', StartDate=%v, EndDate=%v\n", params.SearchTerm, params.StartDate, params.EndDate)

	// For /scan page, only show open jobs - for /jobs page, show all
	// Check if this is called from scan page
	if c.Request.URL.Path == "/scan" || c.Request.URL.Path == "/scan/" {
		params.Status = "Open"
	}

	jobs, err := h.jobRepo.List(params)
	if err != nil {
		// Log the error for debugging
		jobDebugLog("DEBUG: Error loading jobs: %v\n", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// Debug: Log how many jobs were found
	jobDebugLog("DEBUG: Found %d jobs with search term '%s'\n", len(jobs), params.SearchTerm)
	if len(jobs) > 0 {
		jobDebugLog("DEBUG: First job: %+v\n", jobs[0])
	}

	// Get job categories for filter
	jobCategories, _ := h.jobCategoryRepo.List()

	// Get statuses for filter
	statuses, _ := h.statusRepo.List()

	c.HTML(http.StatusOK, "jobs.html", gin.H{
		"title":         "Jobs",
		"jobs":          jobs,
		"params":        params,
		"user":          user,
		"currentPage":   "jobs",
		"jobcategories": jobCategories,
		"statuses":      statuses,
		"timestamp":     "20250820153900", // Force cache refresh
	})
}

func (h *JobHandler) NewJobForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	customers, err := h.customerRepo.List(&models.FilterParams{})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	statuses, err := h.statusRepo.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	jobCategories, err := h.jobCategoryRepo.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// Fetch rental equipment from WarehouseCore
	rentalEquipBySupplier, _ := h.warehouseClient.GetRentalEquipmentBySupplier()

	// Force no-cache headers to prevent template caching issues
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.HTML(http.StatusOK, "job_form.html", gin.H{
		"title":                 "New Job",
		"job":                   &models.Job{},
		"customers":             customers,
		"statuses":              statuses,
		"jobCategories":         jobCategories,
		"user":                  user,
		"rentalEquipBySupplier": rentalEquipBySupplier,
		"jobRentalEquipment":    []models.JobRentalEquipment{},
	})
}

func (h *JobHandler) CreateJob(c *gin.Context) {
	customerID, _ := strconv.ParseUint(c.PostForm("customer_id"), 10, 32)
	statusID, _ := strconv.ParseUint(c.PostForm("status_id"), 10, 32)

	// Validate required fields
	startDateStr := c.PostForm("start_date")
	if startDateStr == "" {
		user, _ := GetCurrentUser(c)
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "New Job",
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Start date is required",
			"user":          user,
		})
		return
	}

	var startDate, endDate *time.Time
	if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
		startDate = &parsed
	} else {
		user, _ := GetCurrentUser(c)
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "New Job",
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Invalid start date format",
			"user":          user,
		})
		return
	}

	// Validate required end date
	endDateStr := c.PostForm("end_date")
	if endDateStr == "" {
		user, _ := GetCurrentUser(c)
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "New Job",
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "End date is required",
			"user":          user,
		})
		return
	}

	if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
		endDate = &parsed
	} else {
		user, _ := GetCurrentUser(c)
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "New Job",
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Invalid end date format",
			"user":          user,
		})
		return
	}

	description := c.PostForm("description")
	discountType := c.PostForm("discount_type")
	if discountType == "" {
		discountType = "amount" // default
	}

	job := models.Job{
		CustomerID:   uint(customerID),
		StatusID:     uint(statusID),
		Description:  &description,
		StartDate:    startDate,
		EndDate:      endDate,
		DiscountType: discountType,
	}

	// Set the creator to the current user
	currentUser, _ := GetCurrentUser(c)
	if currentUser != nil {
		job.CreatedBy = &currentUser.UserID
	}

	if jobCategoryIDStr := c.PostForm("job_category_id"); jobCategoryIDStr != "" {
		if jobCategoryID, err := strconv.ParseUint(jobCategoryIDStr, 10, 32); err == nil {
			jobCatID := uint(jobCategoryID)
			job.JobCategoryID = &jobCatID
		}
	}

	if revenueStr := c.PostForm("revenue"); revenueStr != "" {
		if revenue, err := strconv.ParseFloat(revenueStr, 64); err == nil {
			job.Revenue = revenue
		}
	}

	if discountStr := c.PostForm("discount"); discountStr != "" {
		if discount, err := strconv.ParseFloat(discountStr, 64); err == nil {
			job.Discount = discount
		}
	}

	if err := h.jobRepo.Create(&job); err != nil {
		user, _ := GetCurrentUser(c)
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusInternalServerError, "job_form.html", gin.H{
			"title":         "New Job",
			"job":           &job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         err.Error(),
			"user":          user,
		})
		return
	}

	// Log job creation to history
	if h.jobHistoryService != nil {
		user, _ := GetCurrentUser(c)
		var userID *uint
		if user != nil {
			userID = &user.UserID
		}
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()
		if err := h.jobHistoryService.LogJobCreation(job.JobID, userID, ipAddress, userAgent); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to log job creation: %v\n", err)
		}
	}

	if selectionsStr := c.PostForm("selected_products"); selectionsStr != "" {
		selections, err := parseProductSelectionsFromString(selectionsStr)
		if err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			h.renderJobFormWithError(c, &job, "New Job", "Invalid product selection payload")
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			h.renderJobFormWithError(c, &job, "New Job", err.Error())
			return
		}
	}

	// Process rental equipment selections
	if rentalStr := c.PostForm("selected_rental_equipment"); rentalStr != "" {
		rentalSelections, err := parseRentalEquipmentSelections(rentalStr)
		if err != nil {
			// Log but don't fail - rental equipment is optional
			fmt.Printf("Warning: Failed to parse rental equipment selections: %v\n", err)
		} else if len(rentalSelections) > 0 {
			if err := h.processRentalEquipmentSelections(job.JobID, rentalSelections); err != nil {
				fmt.Printf("Warning: Failed to process rental equipment: %v\n", err)
			}
		}
	}

	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(job.JobID)
	}

	c.Redirect(http.StatusFound, "/jobs")
}

func (h *JobHandler) GetJob(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("id"))
	target := "/jobs"
	if jobID != "" {
		target = fmt.Sprintf("/jobs?editJob=%s", jobID)
	}
	c.Redirect(http.StatusFound, target)
}

func (h *JobHandler) EditJobForm(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("id"))
	target := "/jobs"
	if jobID != "" {
		target = fmt.Sprintf("/jobs?editJob=%s", jobID)
	}
	c.Redirect(http.StatusFound, target)
}

func (h *JobHandler) UpdateJob(c *gin.Context) {
	user, _ := GetCurrentUser(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid job ID", "user": user})
		return
	}

	// Load existing job first
	job, err := h.jobRepo.GetByID(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Job not found", "user": user})
		return
	}

	// Store old job state for history logging
	oldJob := *job

	// Update fields from form
	customerID, _ := strconv.ParseUint(c.PostForm("customer_id"), 10, 32)
	statusID, _ := strconv.ParseUint(c.PostForm("status_id"), 10, 32)
	job.CustomerID = uint(customerID)
	job.StatusID = uint(statusID)

	// Validate required fields
	startDateStr := c.PostForm("start_date")
	if startDateStr == "" {
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "Edit Job",
			"job":           job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Start date is required",
			"user":          user,
		})
		return
	}

	var startDate, endDate *time.Time
	if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
		startDate = &parsed
	} else {
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "Edit Job",
			"job":           job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Invalid start date format",
			"user":          user,
		})
		return
	}

	// Validate required end date
	endDateStr := c.PostForm("end_date")
	if endDateStr == "" {
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "Edit Job",
			"job":           job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "End date is required",
			"user":          user,
		})
		return
	}

	if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
		endDate = &parsed
	} else {
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusBadRequest, "job_form.html", gin.H{
			"title":         "Edit Job",
			"job":           job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         "Invalid end date format",
			"user":          user,
		})
		return
	}
	job.StartDate = startDate
	job.EndDate = endDate

	description := c.PostForm("description")
	job.Description = &description

	discountType := c.PostForm("discount_type")
	if discountType == "" {
		discountType = "amount" // default
	}
	job.DiscountType = discountType

	if jobCategoryIDStr := c.PostForm("job_category_id"); jobCategoryIDStr != "" {
		if jobCategoryID, err := strconv.ParseUint(jobCategoryIDStr, 10, 32); err == nil {
			jobCatID := uint(jobCategoryID)
			job.JobCategoryID = &jobCatID
		}
	}

	if revenueStr := c.PostForm("revenue"); revenueStr != "" {
		if revenue, err := strconv.ParseFloat(revenueStr, 64); err == nil {
			job.Revenue = revenue
		}
	}

	if discountStr := c.PostForm("discount"); discountStr != "" {
		if discount, err := strconv.ParseFloat(discountStr, 64); err == nil {
			job.Discount = discount
		}
	}

	if err := h.jobRepo.Update(job); err != nil {
		customers, _ := h.customerRepo.List(&models.FilterParams{})
		statuses, _ := h.statusRepo.List()
		jobCategories, _ := h.jobCategoryRepo.List()
		c.HTML(http.StatusInternalServerError, "job_form.html", gin.H{
			"title":         "Edit Job",
			"job":           job,
			"customers":     customers,
			"statuses":      statuses,
			"jobCategories": jobCategories,
			"error":         err.Error(),
			"user":          user,
		})
		return
	}

	// Log job update to history
	if h.jobHistoryService != nil {
		var userID *uint
		if user != nil {
			userID = &user.UserID
		}
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()
		if err := h.jobHistoryService.LogJobUpdate(&oldJob, job, userID, ipAddress, userAgent); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to log job update: %v\n", err)
		}
	}

	if selectionsStr := c.PostForm("selected_products"); selectionsStr != "" {
		selections, err := parseProductSelectionsFromString(selectionsStr)
		if err != nil {
			h.renderJobFormWithError(c, job, "Edit Job", "Invalid product selection payload")
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			h.renderJobFormWithError(c, job, "Edit Job", err.Error())
			return
		}
	}

	// Process rental equipment selections
	if rentalStr := c.PostForm("selected_rental_equipment"); rentalStr != "" {
		rentalSelections, err := parseRentalEquipmentSelections(rentalStr)
		if err != nil {
			// Log but don't fail - rental equipment is optional
			fmt.Printf("Warning: Failed to parse rental equipment selections: %v\n", err)
		} else if len(rentalSelections) > 0 {
			if err := h.processRentalEquipmentSelections(job.JobID, rentalSelections); err != nil {
				fmt.Printf("Warning: Failed to process rental equipment: %v\n", err)
			}
		}
	}

	// Only recalculate revenue automatically if no manual revenue was provided
	// This preserves manual revenue entries while still updating when dates change
	if c.PostForm("revenue") == "" {
		h.jobRepo.CalculateAndUpdateRevenue(uint(id))
	} else {
		// If manual revenue was provided, still calculate final_revenue based on discount
		h.jobRepo.UpdateFinalRevenue(uint(id))
	}

	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(uint(id))
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/jobs/%d", id))
}

func (h *JobHandler) DeleteJob(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	if h.calendarSync != nil {
		h.calendarSync.DeleteJobEvent(uint(id))
	}

	if err := h.jobRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func (h *JobHandler) GetJobDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	jobDevices, err := h.jobRepo.GetJobDevices(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Debug logging for device pricing
	jobDebugLog("🔧 DEBUG GetJobDevices: Job %d has %d devices\n", id, len(jobDevices))
	for i, device := range jobDevices {
		customPriceVal := "nil"
		if device.CustomPrice != nil {
			customPriceVal = fmt.Sprintf("%.2f", *device.CustomPrice)
		}

		productPriceVal := "nil"
		if device.Device.Product != nil && device.Device.Product.ItemCostPerDay != nil {
			productPriceVal = fmt.Sprintf("%.2f", *device.Device.Product.ItemCostPerDay)
		}

		jobDebugLog("🔧 DEBUG GetJobDevices[%d]: DeviceID=%s, CustomPrice=%s, ProductPrice=%s\n",
			i, device.DeviceID, customPriceVal, productPriceVal)
	}

	c.JSON(http.StatusOK, gin.H{"devices": jobDevices})
}

func (h *JobHandler) AssignDevice(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.PostForm("device_id")

	price, _ := strconv.ParseFloat(c.PostForm("price"), 64)

	if err := h.jobRepo.AssignDevice(uint(jobID), deviceID, price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device assigned successfully"})
}

func (h *JobHandler) RemoveDevice(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	if err := h.jobRepo.RemoveDevice(uint(jobID), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device removed successfully"})
}

func (h *JobHandler) BulkScanDevices(c *gin.Context) {
	var request models.BulkScanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.jobRepo.BulkAssignDevices(request.JobID, request.DeviceIDs, request.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// API handlers
func (h *JobHandler) ListJobsAPI(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobs, err := h.jobRepo.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// saveRequirements converts product selections to JobProductRequirement rows
// and persists them via the requirement repository (replaces all existing).
func (h *JobHandler) saveRequirements(jobID uint, selections []JobProductSelection) error {
	reqs := make([]models.JobProductRequirement, 0, len(selections))
	for _, s := range selections {
		if s.Quantity <= 0 {
			continue
		}
		reqs = append(reqs, models.JobProductRequirement{
			JobID:     jobID,
			ProductID: s.ProductID,
			Quantity:  s.Quantity,
		})
	}
	return h.requirementRepo.SaveRequirements(jobID, reqs)
}

func (h *JobHandler) CreateJobAPI(c *gin.Context) {
	// Use a map to capture raw JSON data
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create job from request data
	var job models.Job
	// Accept both legacy "customerid" and API "customer_id"
	for _, key := range []string{"customer_id", "customerid"} {
		if customerID, ok := requestData[key]; ok {
			if cid, ok := customerID.(float64); ok && cid > 0 {
				job.CustomerID = uint(cid)
				break
			}
		}
	}
	// Accept both legacy "statusid" and API "status_id"
	job.StatusID = 1 // default: Planung
	for _, key := range []string{"status_id", "statusid"} {
		if statusID, ok := requestData[key]; ok {
			if sid, ok := statusID.(float64); ok && sid > 0 {
				job.StatusID = uint(sid)
				break
			}
		}
	}
	// Job category (accept both snake_case and existing camel-like key)
	if catVal, ok := requestData["job_category_id"]; ok {
		if catStr, ok := catVal.(string); ok {
			if parsed, err := strconv.ParseUint(strings.TrimSpace(catStr), 10, 32); err == nil && parsed > 0 {
				cid := uint(parsed)
				job.JobCategoryID = &cid
			} else {
				job.JobCategoryID = nil
			}
		} else if catNum, ok := catVal.(float64); ok && catNum > 0 {
			cid := uint(catNum)
			job.JobCategoryID = &cid
		}
	} else if catVal, ok := requestData["jobcategoryid"]; ok {
		if catNum, ok := catVal.(float64); ok && catNum > 0 {
			cid := uint(catNum)
			job.JobCategoryID = &cid
		}
	}
	if description, ok := requestData["description"]; ok {
		if desc, ok := description.(string); ok {
			job.Description = &desc
		}
	}
	if discount, ok := requestData["discount"]; ok {
		if d, ok := discount.(float64); ok {
			job.Discount = d
		}
	}
	if discountType, ok := requestData["discount_type"]; ok {
		if dt, ok := discountType.(string); ok {
			job.DiscountType = dt
		}
	}
	if revenue, ok := requestData["revenue"]; ok {
		if r, ok := revenue.(float64); ok {
			job.Revenue = r
		}
	}
	if finalRevenue, ok := requestData["final_revenue"]; ok {
		if fr, ok := finalRevenue.(float64); ok {
			job.FinalRevenue = &fr
		}
	}

	// Handle date fields manually — accept camelCase (frontend) and lowercase (legacy)
	for _, key := range []string{"startDate", "startdate", "start_date"} {
		if startDateStr, ok := requestData[key]; ok {
			if dateStr, ok := startDateStr.(string); ok && dateStr != "" {
				if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
					job.StartDate = &parsed
					break
				}
			}
		}
	}
	for _, key := range []string{"endDate", "enddate", "end_date"} {
		if endDateStr, ok := requestData[key]; ok {
			if dateStr, ok := endDateStr.(string); ok && dateStr != "" {
				if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
					job.EndDate = &parsed
					break
				}
			}
		}
	}

	if venueVal, ok := requestData["venue_id"]; ok {
		if v, ok := venueVal.(float64); ok && v > 0 {
			vid := uint(v)
			job.VenueID = &vid
		}
	}

	// Set the creator to the current user
	currentUser, _ := GetCurrentUser(c)
	if currentUser != nil {
		job.CreatedBy = &currentUser.UserID
	}

	if err := h.jobRepo.Create(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log job creation to history
	if h.jobHistoryService != nil {
		user, _ := GetCurrentUser(c)
		var userID *uint
		if user != nil {
			userID = &user.UserID
		}
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()
		if err := h.jobHistoryService.LogJobCreation(job.JobID, userID, ipAddress, userAgent); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to log job creation: %v\n", err)
		}
	}

	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			_ = h.jobRepo.Delete(job.JobID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(job.JobID)
	}

	c.JSON(http.StatusCreated, job)
}

func (h *JobHandler) GetJobAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := h.jobRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Debug logging to check customer and status data
	jobDebugLog("🔧 DEBUG GetJobAPI: Job %d - CustomerID: %d, StatusID: %d\n", job.JobID, job.CustomerID, job.StatusID)
	jobDebugLog("🔧 DEBUG GetJobAPI: Customer loaded - ID: %d, CompanyName: %v, FirstName: %v, LastName: %v\n",
		job.Customer.CustomerID, job.Customer.CompanyName, job.Customer.FirstName, job.Customer.LastName)
	jobDebugLog("🔧 DEBUG GetJobAPI: Status loaded - ID: %d, Status: %s\n", job.Status.StatusID, job.Status.Status)

	// Debug: Print full JSON being returned
	jsonData, _ := json.MarshalIndent(job, "", "  ")
	jobDebugLog("🔧 DEBUG GetJobAPI: Full JSON response:\n%s\n", string(jsonData))

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) UpdateJobAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Use a map to capture raw JSON data
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing job
	existingJob, err := h.jobRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Store old job state for history logging
	oldJob := *existingJob

	// Create a clean job object without associations to prevent GORM from saving them
	job := models.Job{
		JobID:         existingJob.JobID,
		CustomerID:    existingJob.CustomerID,
		StatusID:      existingJob.StatusID,
		JobCategoryID: existingJob.JobCategoryID,
		Description:   existingJob.Description,
		Discount:      existingJob.Discount,
		DiscountType:  existingJob.DiscountType,
		Revenue:       existingJob.Revenue,
		FinalRevenue:  existingJob.FinalRevenue,
		StartDate:     existingJob.StartDate,
		EndDate:       existingJob.EndDate,
		VenueID:       existingJob.VenueID,
	}
	// Accept both legacy "customerid" and API "customer_id"
	for _, key := range []string{"customer_id", "customerid"} {
		if customerID, ok := requestData[key]; ok {
			if cid, ok := customerID.(float64); ok && cid > 0 {
				job.CustomerID = uint(cid)
				break
			}
		}
	}
	// Accept both legacy "statusid" and API "status_id"
	for _, key := range []string{"status_id", "statusid"} {
		if statusID, ok := requestData[key]; ok {
			if sid, ok := statusID.(float64); ok && sid > 0 {
				job.StatusID = uint(sid)
				break
			}
		}
	}
	if description, ok := requestData["description"]; ok {
		if desc, ok := description.(string); ok {
			job.Description = &desc
		}
	}
	if discount, ok := requestData["discount"]; ok {
		if d, ok := discount.(float64); ok {
			job.Discount = d
		}
	}
	if discountType, ok := requestData["discount_type"]; ok {
		if dt, ok := discountType.(string); ok {
			job.DiscountType = dt
		}
	}
	if revenue, ok := requestData["revenue"]; ok {
		if r, ok := revenue.(float64); ok {
			job.Revenue = r
		}
	}
	if finalRevenue, ok := requestData["final_revenue"]; ok {
		if fr, ok := finalRevenue.(float64); ok {
			job.FinalRevenue = &fr
		}
	}

	// Handle date fields manually — accept camelCase (frontend) and lowercase (legacy)
	for _, key := range []string{"startDate", "startdate", "start_date"} {
		if startDateStr, ok := requestData[key]; ok {
			if dateStr, ok := startDateStr.(string); ok && dateStr != "" {
				if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
					job.StartDate = &parsed
					break
				}
			}
		}
	}
	for _, key := range []string{"endDate", "enddate", "end_date"} {
		if endDateStr, ok := requestData[key]; ok {
			if dateStr, ok := endDateStr.(string); ok && dateStr != "" {
				if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
					job.EndDate = &parsed
					break
				}
			}
		}
	}

	if venueVal, ok := requestData["venue_id"]; ok {
		switch v := venueVal.(type) {
		case float64:
			if v > 0 {
				vid := uint(v)
				job.VenueID = &vid
			} else {
				job.VenueID = nil
			}
		case nil:
			job.VenueID = nil
		}
	}

	if err := h.jobRepo.Update(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log job update to history
	if h.jobHistoryService != nil {
		user, _ := GetCurrentUser(c)
		var userID *uint
		if user != nil {
			userID = &user.UserID
		}
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()
		if err := h.jobHistoryService.LogJobUpdate(&oldJob, &job, userID, ipAddress, userAgent); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to log job update: %v\n", err)
		}
	}

	if selectionsValue, exists := requestData["selected_products"]; exists {
		selections, err := parseProductSelectionsFromInterface(selectionsValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product selection payload"})
			return
		}
		if err := h.saveRequirements(job.JobID, selections); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if h.calendarSync != nil {
		go h.calendarSync.SyncJobEvent(job.JobID)
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) DeleteJobAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	if h.calendarSync != nil {
		h.calendarSync.DeleteJobEvent(uint(id))
	}

	if err := h.jobRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func (h *JobHandler) AssignDeviceAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	var request struct {
		Price float64 `json:"price"`
	}
	_ = c.ShouldBindJSON(&request)

	// Availability check — only when the job has a date range
	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.StartDate != nil && job.EndDate != nil {
		ok, err := h.deviceRepo.IsDeviceAvailableForJob(deviceID, uint(jobID), *job.StartDate, *job.EndDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusConflict, gin.H{"error": "Gerät ist im gewählten Zeitraum bereits vergeben"})
			return
		}
	}

	if err := h.jobRepo.AssignDevice(uint(jobID), deviceID, request.Price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device assigned successfully"})
}

// GetJobRequirementsAPI returns the product requirements for a job,
// enriched with how many devices are currently assigned per product.
func (h *JobHandler) GetJobRequirementsAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	reqs, err := h.requirementRepo.GetByJobID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jobDevices, _ := h.jobRepo.GetJobDevices(uint(jobID))
	assignedPerProduct := make(map[uint]int)
	for _, jd := range jobDevices {
		if jd.Device.ProductID != nil {
			assignedPerProduct[*jd.Device.ProductID]++
		}
	}

	type RequirementRow struct {
		models.JobProductRequirement
		AssignedCount int `json:"assigned_count"`
	}
	rows := make([]RequirementRow, 0, len(reqs))
	for _, r := range reqs {
		rows = append(rows, RequirementRow{
			JobProductRequirement: r,
			AssignedCount:         assignedPerProduct[r.ProductID],
		})
	}

	c.JSON(http.StatusOK, gin.H{"requirements": rows})
}

// GetAvailableDevicesForRequirementAPI returns devices of a given product
// that are available for the job's date range.
func (h *JobHandler) GetAvailableDevicesForRequirementAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}
	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if job.StartDate == nil || job.EndDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job has no date range set"})
		return
	}

	availability, err := h.deviceRepo.GetProductAvailabilityForJob(uint(productID), &job.JobID, job.StartDate, job.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"devices": availability})
}

func (h *JobHandler) RemoveDeviceAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")

	if err := h.jobRepo.RemoveDevice(uint(jobID), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device removed successfully"})
}

func (h *JobHandler) respondWithActiveEditors(c *gin.Context, jobID uint, currentUserID uint) {
	if h.jobEditSessionRepo == nil {
		c.JSON(http.StatusOK, gin.H{"active_editors": []interface{}{}})
		return
	}

	cutoff := time.Now().Add(-jobEditingSessionTTL)
	editors, err := h.jobEditSessionRepo.GetActiveEditors(jobID, currentUserID, cutoff)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load editing sessions"})
		return
	}

	response := make([]gin.H, 0, len(editors))
	for _, editor := range editors {
		response = append(response, gin.H{
			"user_id":      editor.UserID,
			"username":     editor.Username,
			"display_name": editor.DisplayName,
			"last_seen":    editor.LastSeen,
		})
	}

	c.JSON(http.StatusOK, gin.H{"active_editors": response})
}

// StartJobEditingSession registers/refreshes the current user as editing the job.
func (h *JobHandler) StartJobEditingSession(c *gin.Context) {
	if h.jobEditSessionRepo == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "editing session tracking not available"})
		return
	}
	user, exists := GetCurrentUser(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil || jobID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	if exists, err := h.jobRepo.Exists(uint(jobID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify job"})
		return
	} else if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	if err := h.jobEditSessionRepo.UpsertSession(uint(jobID), user.UserID, user.Username, formatUserDisplayName(user)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register editing session"})
		return
	}

	h.respondWithActiveEditors(c, uint(jobID), user.UserID)
}

// StopJobEditingSession removes the current user from the editing session list.
func (h *JobHandler) StopJobEditingSession(c *gin.Context) {
	if h.jobEditSessionRepo == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "editing session tracking not available"})
		return
	}
	user, exists := GetCurrentUser(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil || jobID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	if err := h.jobEditSessionRepo.RemoveSession(uint(jobID), user.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop editing session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// GetJobEditingSessions returns the list of active editors (excluding the current user).
func (h *JobHandler) GetJobEditingSessions(c *gin.Context) {
	if h.jobEditSessionRepo == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "editing session tracking not available"})
		return
	}
	user, exists := GetCurrentUser(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil || jobID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	h.respondWithActiveEditors(c, uint(jobID), user.UserID)
}

func (h *JobHandler) BulkScanDevicesAPI(c *gin.Context) {
	var request models.BulkScanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.jobRepo.BulkAssignDevices(request.JobID, request.DeviceIDs, request.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *JobHandler) UpdateDevicePriceAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: Invalid job ID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")
	jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: JobID=%d, DeviceID=%s\n", jobID, deviceID)

	var request struct {
		Price float64 `json:"price"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: Updating price to %.2f\n", request.Price)

	// Update the device price in the job
	if err := h.jobRepo.UpdateDevicePrice(uint(jobID), deviceID, request.Price); err != nil {
		jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: Repository error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jobDebugLog("🔧 DEBUG UpdateDevicePriceAPI: Success!\n")
	c.JSON(http.StatusOK, gin.H{"message": "Device price updated successfully"})
}

// GetScanBoardData returns the devices for a specific job for the scan board
func (h *JobHandler) GetScanBoardData(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get job to verify it exists
	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Get devices for this job with pack status
	query := `
		SELECT
			jd.deviceID,
			COALESCE(p.name, 'Unknown Product') as productName,
			jd.pack_status as packStatus,
			jd.deviceID as barcodePayload,
			pi.file_path as imageUrl
		FROM job_devices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		LEFT JOIN product_images pi ON p.productID = pi.productID AND pi.is_primary = 1
		WHERE jd.jobID = ?
		ORDER BY p.name, jd.deviceID
	`

	rows, err := h.jobRepo.GetDB().Raw(query, jobID).Rows()
	if err != nil {
		fmt.Printf("Error getting scan board devices: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load devices"})
		return
	}
	defer rows.Close()

	var devices []gin.H
	for rows.Next() {
		var deviceID, productName, packStatus, barcodePayload string
		var imageUrl *string
		err := rows.Scan(&deviceID, &productName, &packStatus, &barcodePayload, &imageUrl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan device data"})
			return
		}

		device := gin.H{
			"deviceid":       deviceID,
			"productName":    productName,
			"packStatus":     packStatus,
			"barcodePayload": barcodePayload,
		}
		if imageUrl != nil && *imageUrl != "" {
			device["imageUrl"] = *imageUrl
		}
		devices = append(devices, device)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobid":       job.JobID,
		"description": job.Description,
		"devices":     devices,
	})
}

// ScanDeviceForPack handles scanning a device for the pack workflow
func (h *JobHandler) ScanDeviceForPack(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var scanReq struct {
		DeviceID       *string `json:"deviceID,omitempty"`
		BarcodePayload *string `json:"barcodePayload,omitempty"`
	}
	if err := c.ShouldBindJSON(&scanReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Determine device ID from request
	var deviceID string
	if scanReq.DeviceID != nil && *scanReq.DeviceID != "" {
		deviceID = *scanReq.DeviceID
	} else if scanReq.BarcodePayload != nil && *scanReq.BarcodePayload != "" {
		deviceID = *scanReq.BarcodePayload
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID or barcode payload required"})
		return
	}

	// Validate that device belongs to this job
	var count int64
	err = h.jobRepo.GetDB().Table("job_devices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Count(&count).Error
	if err != nil {
		fmt.Printf("Error checking device job membership: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device not assigned to this job"})
		return
	}

	// Update pack status to 'packed'
	now := time.Now()
	err = h.jobRepo.GetDB().Table("job_devices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Updates(map[string]interface{}{
			"pack_status": "packed",
			"pack_ts":     now,
		}).Error
	if err != nil {
		fmt.Printf("Error updating pack status: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pack status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Device scanned successfully",
		"deviceid": deviceID,
	})
}

// UpdateDevicePackStatus handles updating pack status for a specific device
func (h *JobHandler) UpdateDevicePackStatus(c *gin.Context) {
	jobIDStr := c.Param("id")
	deviceID := c.Param("deviceId")

	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req struct {
		PackStatus string `json:"pack_status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate pack_status value
	validStatuses := []string{"pending", "packed", "issued", "returned"}
	valid := false
	for _, status := range validStatuses {
		if req.PackStatus == status {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pack_status value"})
		return
	}

	// Validate that device is assigned to this job
	var count int64
	err = h.jobRepo.GetDB().Table("job_devices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Count(&count).Error
	if err != nil {
		fmt.Printf("Error checking device assignment: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not assigned to this job"})
		return
	}

	// Update pack status
	now := time.Now()
	updateData := map[string]interface{}{
		"pack_status": req.PackStatus,
	}

	// Only update pack_ts when marking as packed
	if req.PackStatus == "packed" {
		updateData["pack_ts"] = now
	}

	err = h.jobRepo.GetDB().Table("job_devices").
		Where("jobID = ? AND deviceID = ?", jobID, deviceID).
		Updates(updateData).Error
	if err != nil {
		fmt.Printf("Error updating pack status: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pack status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Pack status updated successfully",
		"deviceid":   deviceID,
		"jobid":      jobID,
		"packStatus": req.PackStatus,
		"updatedAt":  now,
	})
}

// FinishPack handles finishing the pack process
func (h *JobHandler) FinishPack(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var finishReq struct {
		Force bool `json:"force"`
	}
	if err := c.ShouldBindJSON(&finishReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check for missing items
	query := `
		SELECT
			CONCAT(COALESCE(p.name, 'Unknown Product'), ' (', jd.deviceID, ')') as missing_item
		FROM job_devices jd
		LEFT JOIN devices d ON jd.deviceID = d.deviceID
		LEFT JOIN products p ON d.productID = p.productID
		WHERE jd.jobID = ? AND jd.pack_status = 'pending'
		ORDER BY p.name, jd.deviceID
	`

	rows, err := h.jobRepo.GetDB().Raw(query, jobID).Rows()
	if err != nil {
		fmt.Printf("Error getting missing items: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check missing items"})
		return
	}
	defer rows.Close()

	var missing []string
	for rows.Next() {
		var item string
		err := rows.Scan(&item)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan missing items"})
			return
		}
		missing = append(missing, item)
	}

	// If there are missing items and not forcing, return them
	if len(missing) > 0 && !finishReq.Force {
		c.JSON(http.StatusOK, gin.H{
			"success":      false,
			"missingItems": missing,
			"message":      "Some items are not yet packed",
		})
		return
	}

	// Mark all remaining items as packed if forcing
	if finishReq.Force && len(missing) > 0 {
		now := time.Now()
		err = h.jobRepo.GetDB().Table("job_devices").
			Where("jobID = ? AND pack_status = 'pending'", jobID).
			Updates(map[string]interface{}{
				"pack_status": "packed",
				"pack_ts":     now,
			}).Error
		if err != nil {
			fmt.Printf("Error marking all as packed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finish packing"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pack process completed successfully",
	})
}

// AssignPackageToJob handles POST /api/jobs/:id/packages
func (h *JobHandler) AssignPackageToJob(c *gin.Context) {
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var request struct {
		PackageID   int      `json:"package_id" binding:"required"`
		Quantity    uint     `json:"quantity" binding:"required,min=1"`
		CustomPrice *float64 `json:"custom_price"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobPackage, err := h.jobPackageRepo.AssignPackageToJob(jobID, request.PackageID, request.Quantity, request.CustomPrice, user.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Package assigned successfully",
		"job_package": jobPackage,
	})
}

// GetJobPackages handles GET /api/jobs/:id/packages
func (h *JobHandler) GetJobPackages(c *gin.Context) {
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	packages, err := h.jobPackageRepo.GetJobPackagesWithDetails(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"packages": packages,
	})
}

// UpdateJobPackagePrice handles PATCH /api/jobs/packages/:package_id/price
func (h *JobHandler) UpdateJobPackagePrice(c *gin.Context) {
	packageID, err := strconv.ParseUint(c.Param("package_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	var request struct {
		CustomPrice *float64 `json:"custom_price"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.jobPackageRepo.UpdateJobPackagePrice(uint(packageID), request.CustomPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Package price updated successfully",
	})
}

// UpdateJobPackageQuantity handles PATCH /api/jobs/packages/:package_id/quantity
func (h *JobHandler) UpdateJobPackageQuantity(c *gin.Context) {
	packageID, err := strconv.ParseUint(c.Param("package_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	var request struct {
		Quantity uint `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.jobPackageRepo.UpdateJobPackageQuantity(uint(packageID), request.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Package quantity updated successfully",
	})
}

// RemoveJobPackage handles DELETE /api/jobs/packages/:package_id
func (h *JobHandler) RemoveJobPackage(c *gin.Context) {
	packageID, err := strconv.ParseUint(c.Param("package_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	err = h.jobPackageRepo.RemoveJobPackage(uint(packageID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Package removed successfully",
	})
}

// GetJobPackageReservations handles GET /api/jobs/packages/:package_id/reservations
func (h *JobHandler) GetJobPackageReservations(c *gin.Context) {
	packageID, err := strconv.ParseUint(c.Param("package_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	reservations, err := h.jobPackageRepo.GetPackageDeviceReservations(uint(packageID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"reservations": reservations,
	})
}

// CablePlanningResponse represents the cable planning data for a job
type CablePlanningResponse struct {
	JobID             uint               `json:"job_id"`
	TotalWattage      float64            `json:"total_wattage"`
	TotalAmperage16A  float64            `json:"total_amperage_16a"`
	TotalAmperage32A  float64            `json:"total_amperage_32a"`
	DeviceCount       int                `json:"device_count"`
	PowerRequirements []PowerRequirement `json:"power_requirements"`
	CircuitSuggestion string             `json:"circuit_suggestion"`
}

// PowerRequirement represents power needs for a specific product
type PowerRequirement struct {
	ProductID        uint    `json:"product_id"`
	ProductName      string  `json:"product_name"`
	DeviceCount      int     `json:"device_count"`
	WattagePerDevice float64 `json:"wattage_per_device"`
	TotalWattage     float64 `json:"total_wattage"`
}

// GetJobCablePlanning handles GET /api/jobs/:id/cable-planning
// Returns power consumption analysis and cable requirements for a job
func (h *JobHandler) GetJobCablePlanning(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get job with devices
	job, err := h.jobRepo.GetByID(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Get job devices
	jobDevices, err := h.jobRepo.GetJobDevices(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job devices"})
		return
	}

	// Calculate power requirements per product
	productWattage := make(map[uint]*PowerRequirement)
	var totalWattage float64

	for _, jd := range jobDevices {
		if jd.Device.Product == nil {
			continue
		}

		product := jd.Device.Product
		productID := product.ProductID

		if _, exists := productWattage[productID]; !exists {
			wattage := 0.0
			if product.PowerConsumption != nil {
				wattage = *product.PowerConsumption
			}
			productWattage[productID] = &PowerRequirement{
				ProductID:        productID,
				ProductName:      product.Name,
				WattagePerDevice: wattage,
				DeviceCount:      0,
				TotalWattage:     0,
			}
		}

		productWattage[productID].DeviceCount++
		productWattage[productID].TotalWattage = productWattage[productID].WattagePerDevice * float64(productWattage[productID].DeviceCount)
		totalWattage += productWattage[productID].WattagePerDevice
	}

	// Convert map to slice
	powerRequirements := make([]PowerRequirement, 0, len(productWattage))
	for _, pr := range productWattage {
		powerRequirements = append(powerRequirements, *pr)
	}

	// Calculate amperage (assuming 230V single phase)
	// 16A circuit = max 3680W, 32A circuit = max 7360W, 63A = max 14490W
	totalAmperage16A := totalWattage / 230.0
	totalAmperage32A := totalWattage / 400.0 // 3-phase at 400V

	// Suggest circuit type based on total wattage
	var circuitSuggestion string
	switch {
	case totalWattage <= 3680:
		circuitSuggestion = "1x 16A Schuko (Single Phase)"
	case totalWattage <= 7360:
		circuitSuggestion = "1x CEE 32A (3-Phase) or 2x 16A Schuko"
	case totalWattage <= 14490:
		circuitSuggestion = "1x CEE 63A (3-Phase) or 2x CEE 32A"
	default:
		circuitSuggestion = fmt.Sprintf("Multiple power distributions needed (%.0fW total)", totalWattage)
	}

	// Build response
	response := CablePlanningResponse{
		JobID:             job.JobID,
		TotalWattage:      totalWattage,
		TotalAmperage16A:  totalAmperage16A,
		TotalAmperage32A:  totalAmperage32A,
		DeviceCount:       len(jobDevices),
		PowerRequirements: powerRequirements,
		CircuitSuggestion: circuitSuggestion,
	}

	c.JSON(http.StatusOK, response)
}

func (h *JobHandler) ListJobEmployees(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	entries, err := h.jobEmployeeRepo.ListForJob(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *JobHandler) AssignEmployee(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	var body struct {
		EmployeeID uint    `json:"employee_id"`
		Role       *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.EmployeeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee_id required"})
		return
	}
	if err := h.jobEmployeeRepo.Assign(uint(jobID), body.EmployeeID, body.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.calendarSync != nil {
		go h.calendarSync.SyncEmployeeEvent(uint(jobID), body.EmployeeID)
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *JobHandler) RemoveEmployee(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	employeeID, err := strconv.ParseUint(c.Param("employeeId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
		return
	}
	if h.calendarSync != nil {
		h.calendarSync.DeleteEmployeeEvent(uint(jobID), uint(employeeID))
	}
	if err := h.jobEmployeeRepo.Remove(uint(jobID), uint(employeeID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
