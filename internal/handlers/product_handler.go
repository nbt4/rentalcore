package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productRepo *repository.ProductRepository
}

func NewProductHandler(productRepo *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{productRepo: productRepo}
}

// Web interface handlers
func (h *ProductHandler) ListProductsWeb(c *gin.Context) {
	startTime := time.Now()
	log.Printf("🚀 ProductHandler.ListProductsWeb() started")
	
	user, _ := GetCurrentUser(c)
	
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		log.Printf("❌ Error binding query parameters: %v", err)
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=400&message=Bad Request&details=%s", err.Error()))
		return
	}
	
	// Handle search parameter
	searchParam := c.Query("search")
	if searchParam != "" {
		params.SearchTerm = searchParam
	}

	// Handle pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	
	limit := 20 // Products per page
	params.Limit = limit
	params.Offset = (page - 1) * limit
	params.Page = page

	viewType := c.DefaultQuery("view", "list") // Default to list view
	log.Printf("🐛 DEBUG: Product view requested: viewType='%s'", viewType)

	// Get total product count first (without pagination) for proper pagination calculation
	var totalProducts int64
	countQuery := h.productRepo.GetDB().Model(&models.Product{})
	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		countQuery = countQuery.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}
	if params.Category != "" {
		countQuery = countQuery.Where("category = ?", params.Category)
	}
	if err := countQuery.Count(&totalProducts).Error; err != nil {
		log.Printf("❌ Count query error: %v", err)
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}
	
	totalPages := int((totalProducts + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	
	// Get products from database with pagination
	dbStart := time.Now()
	products, err := h.productRepo.List(params)
	dbTime := time.Since(dbStart)
	log.Printf("⏱️  Database query took: %v", dbTime)
	
	if err != nil {
		log.Printf("❌ Database error: %v", err)
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}

	templateStart := time.Now()
	SafeHTML(c, http.StatusOK, "products_standalone.html", gin.H{
		"title":         "Products",
		"products":      products,
		"params":        params,
		"user":          user,
		"viewType":      viewType,
		"currentPage":   "products",
		"pageNumber":    page,
		"hasNextPage":   page < totalPages,
		"totalPages":    totalPages,
		"totalProducts": int(totalProducts),
	})
	
	templateTime := time.Since(templateStart)
	totalTime := time.Since(startTime)
	log.Printf("⏱️  Template rendering took: %v", templateTime)
	log.Printf("🏁 ProductHandler.ListProductsWeb() completed in %v", totalTime)
}

func (h *ProductHandler) NewProductForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	// Get categories for the form
	categories, err := h.productRepo.GetAllCategories()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.HTML(http.StatusOK, "product_form.html", gin.H{
		"title":      "New Product",
		"product":    &models.Product{},
		"categories": categories,
		"user":       user,
	})
}

// API handlers (existing)
func (h *ProductHandler) ListProducts(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	products, err := h.productRepo.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (h *ProductHandler) GetProductAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}
	
	product, err := h.productRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (h *ProductHandler) CreateProductAPI(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		log.Printf("❌ Error binding product JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid product data: %v", err)})
		return
	}

	log.Printf("📦 Creating product: %+v", product)

	if err := h.productRepo.Create(&product); err != nil {
		log.Printf("❌ Error creating product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"product": product})
}

func (h *ProductHandler) UpdateProductAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		log.Printf("❌ Error binding product JSON for update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid product data: %v", err)})
		return
	}

	log.Printf("📦 Updating product %d: %+v", id, product)

	product.ProductID = uint(id)
	if err := h.productRepo.Update(&product); err != nil {
		log.Printf("❌ Error updating product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// GetSubcategoriesAPI returns all subcategories
func (h *ProductHandler) GetSubcategoriesAPI(c *gin.Context) {
	var subcategories []models.Subcategory
	if err := h.productRepo.GetAllSubcategories(&subcategories); err != nil {
		log.Printf("❌ Error fetching subcategories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subcategories"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"subcategories": subcategories})
}

// GetSubbiercategoriesAPI returns all subbiercategories
func (h *ProductHandler) GetSubbiercategoriesAPI(c *gin.Context) {
	var subbiercategories []models.Subbiercategory
	if err := h.productRepo.GetAllSubbiercategories(&subbiercategories); err != nil {
		log.Printf("❌ Error fetching subbiercategories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subbiercategories"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"subbiercategories": subbiercategories})
}

// GetBrandsAPI returns all brands
func (h *ProductHandler) GetBrandsAPI(c *gin.Context) {
	var brands []models.Brand
	if err := h.productRepo.GetAllBrands(&brands); err != nil {
		log.Printf("❌ Error fetching brands: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brands"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"brands": brands})
}

// GetManufacturersAPI returns all manufacturers
func (h *ProductHandler) GetManufacturersAPI(c *gin.Context) {
	var manufacturers []models.Manufacturer
	if err := h.productRepo.GetAllManufacturers(&manufacturers); err != nil {
		log.Printf("❌ Error fetching manufacturers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manufacturers"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"manufacturers": manufacturers})
}

func (h *ProductHandler) DeleteProductAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) GetCategoriesAPI(c *gin.Context) {
	categories, err := h.productRepo.GetAllCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}