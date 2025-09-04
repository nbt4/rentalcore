package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobRepo         *repository.JobRepository
	deviceRepo      *repository.DeviceRepository
	customerRepo    *repository.CustomerRepository
	statusRepo      *repository.StatusRepository
	jobCategoryRepo *repository.JobCategoryRepository
}

func NewJobHandler(jobRepo *repository.JobRepository, deviceRepo *repository.DeviceRepository, customerRepo *repository.CustomerRepository, statusRepo *repository.StatusRepository, jobCategoryRepo *repository.JobCategoryRepository) *JobHandler {
	return &JobHandler{
		jobRepo:         jobRepo,
		deviceRepo:      deviceRepo,
		customerRepo:    customerRepo,
		statusRepo:      statusRepo,
		jobCategoryRepo: jobCategoryRepo,
	}
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
	fmt.Printf("DEBUG Job Handler: All query params: %+v\n", c.Request.URL.Query())
	
	// Manual parameter extraction to ensure search works
	searchParam := c.Query("search")
	fmt.Printf("DEBUG Job Handler: Raw search parameter: '%s'\n", searchParam)
	if searchParam != "" {
		params.SearchTerm = searchParam
		fmt.Printf("DEBUG Job Handler: Search parameter SET to: '%s'\n", searchParam)
	}
	
	// DEBUG: Log params after binding
	fmt.Printf("DEBUG Job Handler: Final params: SearchTerm='%s', StartDate=%v, EndDate=%v\n", params.SearchTerm, params.StartDate, params.EndDate)

	// For /scan page, only show open jobs - for /jobs page, show all
	// Check if this is called from scan page
	if c.Request.URL.Path == "/scan" || c.Request.URL.Path == "/scan/" {
		params.Status = "Open"
	}
	
	jobs, err := h.jobRepo.List(params)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("DEBUG: Error loading jobs: %v\n", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// Debug: Log how many jobs were found
	fmt.Printf("DEBUG: Found %d jobs with search term '%s'\n", len(jobs), params.SearchTerm)
	if len(jobs) > 0 {
		fmt.Printf("DEBUG: First job: %+v\n", jobs[0])
	}

	c.HTML(http.StatusOK, "jobs.html", gin.H{
		"title":       "Jobs",
		"jobs":        jobs,
		"params":      params,
		"user":        user,
		"currentPage": "jobs",
		"timestamp":   "20250820153900", // Force cache refresh
	})
}

func (h *JobHandler) NewJobForm(c *gin.Context) {
	// Only allow AJAX requests (from modal), block direct browser access
	if c.GetHeader("X-Requested-With") != "XMLHttpRequest" && c.GetHeader("Accept") != "application/json" {
		// Check if this is a fetch request (which doesn't set X-Requested-With)
		acceptHeader := c.GetHeader("Accept")
		if !strings.Contains(acceptHeader, "text/html") || strings.Contains(c.GetHeader("User-Agent"), "fetch") {
			// This looks like a fetch request from the modal, allow it
		} else {
			// Direct browser access - redirect to jobs list
			c.Redirect(http.StatusFound, "/jobs")
			return
		}
	}
	
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

	c.HTML(http.StatusOK, "job_form.html", gin.H{
		"title":        "New Job",
		"job":          &models.Job{},
		"customers":    customers,
		"statuses":     statuses,
		"jobCategories": jobCategories,
		"user":         user,
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
			"title":        "New Job",
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Start date is required",
			"user":         user,
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
			"title":        "New Job",
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Invalid start date format",
			"user":         user,
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
			"title":        "New Job",
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "End date is required",
			"user":         user,
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
			"title":        "New Job",
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Invalid end date format",
			"user":         user,
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
			"title":        "New Job",
			"job":          &job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        err.Error(),
			"user":         user,
		})
		return
	}

	c.Redirect(http.StatusFound, "/jobs")
}

func (h *JobHandler) GetJob(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid job ID", "user": user})
		return
	}

	job, err := h.jobRepo.GetByID(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Job not found", "user": user})
		return
	}

	jobDevices, err := h.jobRepo.GetJobDevices(uint(id))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// Group devices by product and calculate pricing
	productGroups := make(map[string]*ProductGroup)
	totalDevices := len(jobDevices)
	totalValue := 0.0

	for _, jd := range jobDevices {
		if jd.Device.Product == nil {
			continue
		}

		productName := jd.Device.Product.Name
		if _, exists := productGroups[productName]; !exists {
			productGroups[productName] = &ProductGroup{
				Product: jd.Device.Product,
				Devices: []models.JobDevice{},
			}
		}

		// Calculate effective price (custom price if set, otherwise default product price)
		var effectivePrice float64
		if jd.CustomPrice != nil && *jd.CustomPrice > 0 {
			effectivePrice = *jd.CustomPrice
		} else if jd.Device.Product.ItemCostPerDay != nil {
			effectivePrice = *jd.Device.Product.ItemCostPerDay
		}

		// Create a copy of the job device with calculated price for display
		jdCopy := jd
		jdCopy.CustomPrice = &effectivePrice

		productGroups[productName].Devices = append(productGroups[productName].Devices, jdCopy)
		productGroups[productName].Count = len(productGroups[productName].Devices)
		productGroups[productName].TotalValue += effectivePrice
		totalValue += effectivePrice
	}

	c.HTML(http.StatusOK, "job_detail.html", gin.H{
		"title":         "Job Details",
		"job":           job,
		"jobDevices":    jobDevices,
		"productGroups": productGroups,
		"totalDevices":  totalDevices,
		"totalValue":    totalValue,
		"user":          user,
	})
}

func (h *JobHandler) EditJobForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid job ID", "user": user})
		return
	}


	job, err := h.jobRepo.GetByID(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Job not found", "user": user})
		return
	}

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


	c.HTML(http.StatusOK, "job_form.html", gin.H{
		"title":        "Edit Job",
		"job":          job,
		"customers":    customers,
		"statuses":     statuses,
		"jobCategories": jobCategories,
		"user":         user,
	})
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
			"title":        "Edit Job",
			"job":          job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Start date is required",
			"user":         user,
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
			"title":        "Edit Job",
			"job":          job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Invalid start date format",
			"user":         user,
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
			"title":        "Edit Job",
			"job":          job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "End date is required",
			"user":         user,
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
			"title":        "Edit Job",
			"job":          job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        "Invalid end date format",
			"user":         user,
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
			"title":        "Edit Job",
			"job":          job,
			"customers":    customers,
			"statuses":     statuses,
			"jobCategories": jobCategories,
			"error":        err.Error(),
			"user":         user,
		})
		return
	}

	// Only recalculate revenue automatically if no manual revenue was provided
	// This preserves manual revenue entries while still updating when dates change
	if c.PostForm("revenue") == "" {
		h.jobRepo.CalculateAndUpdateRevenue(uint(id))
	} else {
		// If manual revenue was provided, still calculate final_revenue based on discount
		h.jobRepo.UpdateFinalRevenue(uint(id))
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/jobs/%d", id))
}

func (h *JobHandler) DeleteJob(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
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
	fmt.Printf("🔧 DEBUG GetJobDevices: Job %d has %d devices\n", id, len(jobDevices))
	for i, device := range jobDevices {
		customPriceVal := "nil"
		if device.CustomPrice != nil {
			customPriceVal = fmt.Sprintf("%.2f", *device.CustomPrice)
		}
		
		productPriceVal := "nil"
		if device.Device.Product != nil && device.Device.Product.ItemCostPerDay != nil {
			productPriceVal = fmt.Sprintf("%.2f", *device.Device.Product.ItemCostPerDay)
		}
		
		fmt.Printf("🔧 DEBUG GetJobDevices[%d]: DeviceID=%s, CustomPrice=%s, ProductPrice=%s\n", 
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

func (h *JobHandler) CreateJobAPI(c *gin.Context) {
	// Use a map to capture raw JSON data
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create job from request data
	var job models.Job
	if customerID, ok := requestData["customerID"]; ok {
		if cid, ok := customerID.(float64); ok {
			job.CustomerID = uint(cid)
		}
	}
	if statusID, ok := requestData["statusID"]; ok {
		if sid, ok := statusID.(float64); ok {
			job.StatusID = uint(sid)
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

	// Handle date fields manually
	if startDateStr, ok := requestData["startDate"]; ok {
		if dateStr, ok := startDateStr.(string); ok && dateStr != "" {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				job.StartDate = &parsed
			}
		}
	}
	if endDateStr, ok := requestData["endDate"]; ok {
		if dateStr, ok := endDateStr.(string); ok && dateStr != "" {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				job.EndDate = &parsed
			}
		}
	}

	if err := h.jobRepo.Create(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
	fmt.Printf("🔧 DEBUG GetJobAPI: Job %d - CustomerID: %d, StatusID: %d\n", job.JobID, job.CustomerID, job.StatusID)
	fmt.Printf("🔧 DEBUG GetJobAPI: Customer loaded - ID: %d, CompanyName: %v, FirstName: %v, LastName: %v\n", 
		job.Customer.CustomerID, job.Customer.CompanyName, job.Customer.FirstName, job.Customer.LastName)
	fmt.Printf("🔧 DEBUG GetJobAPI: Status loaded - ID: %d, Status: %s\n", job.Status.StatusID, job.Status.Status)
	
	// Debug: Print full JSON being returned
	jsonData, _ := json.MarshalIndent(job, "", "  ")
	fmt.Printf("🔧 DEBUG GetJobAPI: Full JSON response:\n%s\n", string(jsonData))

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
	}
	if customerID, ok := requestData["customerID"]; ok {
		if cid, ok := customerID.(float64); ok {
			job.CustomerID = uint(cid)
		}
	}
	if statusID, ok := requestData["statusID"]; ok {
		if sid, ok := statusID.(float64); ok {
			job.StatusID = uint(sid)
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

	// Handle date fields manually
	if startDateStr, ok := requestData["startDate"]; ok {
		if dateStr, ok := startDateStr.(string); ok && dateStr != "" {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				job.StartDate = &parsed
			}
		}
	}
	if endDateStr, ok := requestData["endDate"]; ok {
		if dateStr, ok := endDateStr.(string); ok && dateStr != "" {
			if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
				job.EndDate = &parsed
			}
		}
	}

	if err := h.jobRepo.Update(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle device assignments if selected_devices is provided
	if selectedDevicesStr, ok := requestData["selected_devices"]; ok {
		if deviceStr, ok := selectedDevicesStr.(string); ok && deviceStr != "" {
			// Parse selected devices
			selectedDevices := strings.Split(deviceStr, ",")
			
			// Get current job devices
			currentDevices, err := h.jobRepo.GetJobDevices(uint(id))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current devices"})
				return
			}
			
			// Create sets for comparison
			currentDeviceIDs := make(map[string]bool)
			for _, device := range currentDevices {
				currentDeviceIDs[device.DeviceID] = true
			}
			
			newDeviceIDs := make(map[string]bool)
			for _, deviceID := range selectedDevices {
				if deviceID != "" {
					newDeviceIDs[deviceID] = true
				}
			}
			
			// Remove devices that are no longer selected
			for deviceID := range currentDeviceIDs {
				if !newDeviceIDs[deviceID] {
					if err := h.jobRepo.UnassignDevice(uint(id), deviceID); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unassign device " + deviceID})
						return
					}
				}
			}
			
			// Add new devices
			for deviceID := range newDeviceIDs {
				if !currentDeviceIDs[deviceID] {
					if err := h.jobRepo.AssignDevice(uint(id), deviceID, 0.0); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign device " + deviceID})
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) DeleteJobAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
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
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.jobRepo.AssignDevice(uint(jobID), deviceID, request.Price); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device assigned successfully"})
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
		fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: Invalid job ID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	deviceID := c.Param("deviceId")
	fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: JobID=%d, DeviceID=%s\n", jobID, deviceID)
	
	var request struct {
		Price float64 `json:"price"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: Updating price to %.2f\n", request.Price)

	// Update the device price in the job
	if err := h.jobRepo.UpdateDevicePrice(uint(jobID), deviceID, request.Price); err != nil {
		fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: Repository error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🔧 DEBUG UpdateDevicePriceAPI: Success!\n")
	c.JSON(http.StatusOK, gin.H{"message": "Device price updated successfully"})
}