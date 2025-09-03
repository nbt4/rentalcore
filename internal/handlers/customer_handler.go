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
)

type CustomerHandler struct {
	customerRepo *repository.CustomerRepository
}

func NewCustomerHandler(customerRepo *repository.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{customerRepo: customerRepo}
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
	user, _ := GetCurrentUser(c)
	
	c.HTML(http.StatusOK, "customer_form.html", gin.H{
		"title":    "New Customer",
		"customer": &models.Customer{},
		"user":     user,
	})
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	// Debug: Print all form data
	fmt.Printf("🚨 DEBUG: Customer creation called!\n")
	fmt.Printf("🚨 DEBUG: HTTP Method: %s\n", c.Request.Method)
	fmt.Printf("🚨 DEBUG: Content-Type: %s\n", c.ContentType())
	fmt.Printf("🚨 DEBUG: All form fields:\n")
	
	// Parse form first
	c.Request.ParseForm()
	for key, values := range c.Request.PostForm {
		fmt.Printf("   %s: %v\n", key, values)
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
	fmt.Printf("🔧 DEBUG: Creating customer with parsed data:\n")
	fmt.Printf("   CompanyName: '%s'\n", companyName)
	fmt.Printf("   FirstName: '%s'\n", firstName)
	fmt.Printf("   LastName: '%s'\n", lastName)
	fmt.Printf("   Email: '%s'\n", email)
	fmt.Printf("   PhoneNumber: '%s'\n", phoneNumber)
	fmt.Printf("   CustomerType: '%s'\n", customerType)
	
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

	fmt.Printf("🔧 DEBUG: Calling customerRepo.Create()\n")
	if err := h.customerRepo.Create(&customer); err != nil {
		fmt.Printf("❌ DEBUG: Customer creation failed: %v\n", err)
		user, _ := GetCurrentUser(c)
		c.HTML(http.StatusInternalServerError, "customer_form.html", gin.H{
			"title":    "New Customer",
			"customer": &customer,
			"error":    err.Error(),
			"user":     user,
		})
		return
	}

	fmt.Printf("✅ DEBUG: Customer creation succeeded, ID: %d\n", customer.CustomerID)
	
	// Add a simple success page instead of redirect for debugging
	c.HTML(http.StatusOK, "customers.html", gin.H{
		"title": "Success!",
		"message": fmt.Sprintf("Customer created successfully with ID: %d", customer.CustomerID),
	})
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
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

	c.Redirect(http.StatusFound, "/customers")
}

func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := h.customerRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	customers, err := h.customerRepo.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}

func (h *CustomerHandler) CreateCustomerAPI(c *gin.Context) {
	fmt.Printf("🚨 DEBUG API: CreateCustomerAPI called\n")
	fmt.Printf("🚨 DEBUG API: Content-Type: %s\n", c.ContentType())
	
	// Debug: Print raw request body
	bodyBytes, _ := c.GetRawData()
	fmt.Printf("🚨 DEBUG API: Raw request body: %s\n", string(bodyBytes))
	
	// Reset the request body so it can be read again
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	
	var customer models.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		fmt.Printf("❌ DEBUG API: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("✅ DEBUG API: Parsed customer: %+v\n", customer)

	if err := h.customerRepo.Create(&customer); err != nil {
		fmt.Printf("❌ DEBUG API: Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🎉 DEBUG API: Customer created successfully with ID: %d\n", customer.CustomerID)
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

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) DeleteCustomerAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := h.customerRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}