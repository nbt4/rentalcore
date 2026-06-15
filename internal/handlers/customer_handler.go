package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"

	"go-barcode-webapp/internal/logger"
)

// SyncServiceInterface erlaubt nil-Check ohne Import-Zyklus.
type SyncServiceInterface interface {
	PushCreate(customer *models.Customer) error
	PushUpdate(customer *models.Customer) error
	PushDelete(customer *models.Customer)
}

type CustomerHandler struct {
	customerRepo *repository.CustomerRepository
	syncService  SyncServiceInterface
}

func NewCustomerHandler(customerRepo *repository.CustomerRepository, syncService SyncServiceInterface) *CustomerHandler {
	return &CustomerHandler{customerRepo: customerRepo, syncService: syncService}
}

func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	// Manual parameter extraction to ensure search works
	searchParam := c.Query("search")
	if searchParam != "" {
		params.SearchTerm = searchParam
	}

	customers, err := h.customerRepo.List(params)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.HTML(http.StatusOK, "customers.html", gin.H{
		"title":       "Customers",
		"customers":   customers,
		"params":      params,
		"user":        user,
		"currentPage": "customers",
	})
}

func (h *CustomerHandler) NewCustomerForm(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/customers")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/customers")
		return
	}

	user, _ := GetCurrentUser(c)
	
	c.HTML(http.StatusOK, "customer_form.html", gin.H{
		"title":    "New Customer",
		"customer": &models.Customer{},
		"user":     user,
	})
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	// Debug: Print all form data
	logger.LogWarn("🚨 DEBUG: Customer creation called!\n")
	logger.LogWarn("🚨 DEBUG: HTTP Method: %s\n", c.Request.Method)
	logger.LogWarn("🚨 DEBUG: Content-Type: %s\n", c.ContentType())
	logger.LogWarn("🚨 DEBUG: All form fields:\n")
	
	// Parse form first
	c.Request.ParseForm()
	for key, values := range c.Request.PostForm {
		logger.LogWarn("   %s: %v\n", key, values)
	}
	
	companyName := c.PostForm("company_name")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	email := c.PostForm("email")
	phoneNumber := c.PostForm("phone_number")
	street := c.PostForm("street")
	houseNumber := c.PostForm("house_number")
	zip := c.PostForm("zip")
	city := c.PostForm("city")
	federalState := c.PostForm("federal_state")
	country := c.PostForm("country")
	customerType := c.PostForm("customer_type")
	notes := c.PostForm("notes")
	
	// Debug logging
	logger.LogWarn("🔧 DEBUG: Creating customer with parsed data:\n")
	logger.LogWarn("   CompanyName: '%s'\n", companyName)
	logger.LogWarn("   FirstName: '%s'\n", firstName)
	logger.LogWarn("   LastName: '%s'\n", lastName)
	logger.LogWarn("   Email: '%s'\n", email)
	logger.LogWarn("   PhoneNumber: '%s'\n", phoneNumber)
	logger.LogWarn("   CustomerType: '%s'\n", customerType)
	
	customer := models.Customer{
		CompanyName:  &companyName,
		FirstName:    &firstName,
		LastName:     &lastName,
		Email:        &email,
		PhoneNumber:  &phoneNumber,
		Street:       &street,
		HouseNumber:  &houseNumber,
		ZIP:          &zip,
		City:         &city,
		FederalState: &federalState,
		Country:      &country,
		CustomerType: &customerType,
		Notes:        &notes,
	}

	logger.LogWarn("🔧 DEBUG: Calling customerRepo.Create()\n")
	if err := h.customerRepo.Create(&customer); err != nil {
		logger.LogWarn("❌ DEBUG: Customer creation failed: %v\n", err)
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "customer_form.html", gin.H{
			"title":    "New Customer",
			"customer": &customer,
			"error":    err.Error(),
			"user":     user,
		})
		return
	}

	if h.syncService != nil {
		if err := h.syncService.PushCreate(&customer); err != nil {
			logger.LogInfo("M365 sync PushCreate failed: %v", err)
		}
	}

	logger.LogWarn("✅ DEBUG: Customer creation succeeded, ID: %d\n", customer.CustomerID)
	
	// Add a simple success page instead of redirect for debugging
	c.HTML(http.StatusOK, "customers.html", gin.H{
		"title": "Success!",
		"message": fmt.Sprintf("Customer created successfully with ID: %d", customer.CustomerID),
	})
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/customers")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/customers")
		return
	}

	user, _ := GetCurrentUser(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid customer ID", "user": user})
		return
	}

	customer, err := h.customerRepo.GetByID(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Customer not found", "user": user})
		return
	}

	c.HTML(http.StatusOK, "customer_detail.html", gin.H{
		"customer": customer,
		"user":     user,
	})
}

func (h *CustomerHandler) EditCustomerForm(c *gin.Context) {
	// Only allow fetch requests from modals, block direct browser access
	acceptHeader := c.GetHeader("Accept")
	xRequestedWith := c.GetHeader("X-Requested-With")
	
	// Block direct browser access - only allow modal/fetch requests
	if xRequestedWith != "XMLHttpRequest" && !strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "text/html") {
		c.Redirect(http.StatusFound, "/customers")
		return
	}
	
	// If it's a direct browser request (Accept: text/html without XMLHttpRequest), redirect
	if strings.Contains(acceptHeader, "text/html") && xRequestedWith != "XMLHttpRequest" {
		c.Redirect(http.StatusFound, "/customers")
		return
	}

	user, _ := GetCurrentUser(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid customer ID", "user": user})
		return
	}

	customer, err := h.customerRepo.GetByID(uint(id))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Customer not found", "user": user})
		return
	}

	c.HTML(http.StatusOK, "customer_form.html", gin.H{
		"title":    "Edit Customer",
		"customer": customer,
		"user":     user,
	})
}

func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid customer ID", "user": user})
		return
	}

	companyName := c.PostForm("company_name")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	email := c.PostForm("email")
	phoneNumber := c.PostForm("phone_number")
	street := c.PostForm("street")
	houseNumber := c.PostForm("house_number")
	zip := c.PostForm("zip")
	city := c.PostForm("city")
	federalState := c.PostForm("federal_state")
	country := c.PostForm("country")
	customerType := c.PostForm("customer_type")
	notes := c.PostForm("notes")
	
	customer := models.Customer{
		CustomerID:   uint(id),
		CompanyName:  &companyName,
		FirstName:    &firstName,
		LastName:     &lastName,
		Email:        &email,
		PhoneNumber:  &phoneNumber,
		Street:       &street,
		HouseNumber:  &houseNumber,
		ZIP:          &zip,
		City:         &city,
		FederalState: &federalState,
		Country:      &country,
		CustomerType: &customerType,
		Notes:        &notes,
	}

	if err := h.customerRepo.Update(&customer); err != nil {
		c.HTML(http.StatusInternalServerError, "customer_form.html", gin.H{
			"title":    "Edit Customer",
			"customer": &customer,
			"error":    err.Error(),
			"user":     user,
		})
		return
	}

	if h.syncService != nil {
		if saved, err := h.customerRepo.GetByID(customer.CustomerID); err == nil {
			if err := h.syncService.PushUpdate(saved); err != nil {
				logger.LogInfo("M365 sync PushUpdate failed: %v", err)
			}
		}
	}

	c.Redirect(http.StatusFound, "/customers")
}

func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var customerForSync *models.Customer
	if h.syncService != nil {
		customerForSync, _ = h.customerRepo.GetByID(uint(id))
	}

	if err := h.customerRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.syncService != nil && customerForSync != nil {
		h.syncService.PushDelete(customerForSync)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}

// API handlers
func (h *CustomerHandler) ListCustomersAPI(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := c.Query("role")
	customers, err := h.customerRepo.ListByRole(params, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

func (h *CustomerHandler) CreateCustomerAPI(c *gin.Context) {
	logger.LogWarn("🚨 DEBUG API: CreateCustomerAPI called\n")
	logger.LogWarn("🚨 DEBUG API: Content-Type: %s\n", c.ContentType())
	
	// Debug: Print raw request body
	bodyBytes, _ := c.GetRawData()
	logger.LogWarn("🚨 DEBUG API: Raw request body: %s\n", string(bodyBytes))
	
	// Reset the request body so it can be read again
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	
	var customer models.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		logger.LogWarn("❌ DEBUG API: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.LogWarn("✅ DEBUG API: Parsed customer: %+v\n", customer)

	if err := h.customerRepo.Create(&customer); err != nil {
		logger.LogWarn("❌ DEBUG API: Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.syncService != nil {
		if err := h.syncService.PushCreate(&customer); err != nil {
			logger.LogInfo("M365 sync PushCreate failed: %v", err)
		}
	}

	logger.LogWarn("🎉 DEBUG API: Customer created successfully with ID: %d\n", customer.CustomerID)
	c.JSON(http.StatusCreated, customer)
}

func (h *CustomerHandler) GetCustomerAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := h.customerRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) UpdateCustomerAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var customer models.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer.CustomerID = uint(id)
	if err := h.customerRepo.Update(&customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.syncService != nil {
		if saved, err := h.customerRepo.GetByID(customer.CustomerID); err == nil {
			if err := h.syncService.PushUpdate(saved); err != nil {
				logger.LogInfo("M365 sync PushUpdate failed: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) DeleteCustomerAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var customerForSync *models.Customer
	if h.syncService != nil {
		customerForSync, _ = h.customerRepo.GetByID(uint(id))
	}

	if err := h.customerRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.syncService != nil && customerForSync != nil {
		h.syncService.PushDelete(customerForSync)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}