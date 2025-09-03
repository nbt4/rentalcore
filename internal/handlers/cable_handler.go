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

type CableHandler struct {
	cableRepo *repository.CableRepository
}

func NewCableHandler(cableRepo *repository.CableRepository) *CableHandler {
	return &CableHandler{cableRepo: cableRepo}
}

// Web interface handlers
func (h *CableHandler) ListCablesWeb(c *gin.Context) {
	startTime := time.Now()
	log.Printf("🚀 CableHandler.ListCablesWeb() started")
	
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
	
	limit := 20 // Cables per page
	params.Limit = limit
	params.Offset = (page - 1) * limit
	params.Page = page

	viewType := c.DefaultQuery("view", "list") // Default to list view
	log.Printf("🐛 DEBUG: Cable view requested: viewType='%s'", viewType)

	// Get cables from database (grouped by specifications)
	dbStart := time.Now()
	cableGroups, err := h.cableRepo.ListGrouped(params)
	dbTime := time.Since(dbStart)
	log.Printf("⏱️  Database query took: %v", dbTime)
	
	if err != nil {
		log.Printf("❌ Database error: %v", err)
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/error?code=500&message=Database Error&details=%s", err.Error()))
		return
	}
	
	// Get total cable count for pagination
	totalCables, err := h.cableRepo.GetTotalCount()
	if err != nil {
		log.Printf("❌ Error getting total cable count: %v", err)
		totalCables = 0
	}
	
	totalPages := (totalCables + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	templateStart := time.Now()
	SafeHTML(c, http.StatusOK, "cables_standalone.html", gin.H{
		"title":        "Cables",
		"cableGroups":  cableGroups,
		"params":       params,
		"user":         user,
		"viewType":     viewType,
		"currentPage":  "cables",
		"pageNumber":   page,
		"hasNextPage":  page < totalPages,
		"totalPages":   totalPages,
		"totalCables":  totalCables,
	})
	
	templateTime := time.Since(templateStart)
	totalTime := time.Since(startTime)
	log.Printf("⏱️  Template rendering took: %v", templateTime)
	log.Printf("🏁 CableHandler.ListCablesWeb() completed in %v", totalTime)
}

func (h *CableHandler) NewCableForm(c *gin.Context) {
	user, _ := GetCurrentUser(c)
	
	// Get cable types and connectors for the form
	types, err := h.cableRepo.GetAllCableTypes()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}
	
	connectors, err := h.cableRepo.GetAllCableConnectors()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error(), "user": user})
		return
	}

	c.HTML(http.StatusOK, "cable_form.html", gin.H{
		"title":      "New Cable",
		"cable":      &models.Cable{},
		"types":      types,
		"connectors": connectors,
		"user":       user,
	})
}

func (h *CableHandler) CreateCable(c *gin.Context) {
	log.Printf("🔥 CREATE CABLE HANDLER CALLED")
	
	// Parse form values
	connector1Str := c.PostForm("connector1")
	connector2Str := c.PostForm("connector2")
	typeStr := c.PostForm("type")
	lengthStr := c.PostForm("length")
	mm2Str := c.PostForm("mm2")
	amountStr := c.PostForm("amount")
	
	log.Printf("📝 Form values: connector1='%s', connector2='%s', type='%s', length='%s', mm2='%s', amount='%s'", 
		connector1Str, connector2Str, typeStr, lengthStr, mm2Str, amountStr)
	
	// Parse required fields
	connector1, err := strconv.Atoi(connector1Str)
	if err != nil {
		log.Printf("❌ Invalid connector1: %v", err)
		h.renderCableFormWithError(c, "Invalid connector 1 value", nil)
		return
	}
	
	connector2, err := strconv.Atoi(connector2Str)
	if err != nil {
		log.Printf("❌ Invalid connector2: %v", err)
		h.renderCableFormWithError(c, "Invalid connector 2 value", nil)
		return
	}
	
	cableType, err := strconv.Atoi(typeStr)
	if err != nil {
		log.Printf("❌ Invalid type: %v", err)
		h.renderCableFormWithError(c, "Invalid cable type value", nil)
		return
	}
	
	length, err := strconv.ParseFloat(lengthStr, 64)
	if err != nil {
		log.Printf("❌ Invalid length: %v", err)
		h.renderCableFormWithError(c, "Invalid length value", nil)
		return
	}
	
	var mm2 *float64
	if mm2Str != "" {
		parsedMM2, err := strconv.ParseFloat(mm2Str, 64)
		if err != nil {
			log.Printf("❌ Invalid mm2: %v", err)
			h.renderCableFormWithError(c, "Invalid mm² value", nil)
			return
		}
		mm2 = &parsedMM2
	}
	
	// Parse amount (default to 1 if not provided)
	amount := 1
	if amountStr != "" {
		amount, err = strconv.Atoi(amountStr)
		if err != nil || amount < 1 {
			log.Printf("❌ Invalid amount: %v", err)
			h.renderCableFormWithError(c, "Invalid amount value", nil)
			return
		}
	}
	
	// Create cables based on amount
	var createdCables []models.Cable
	var createdIDs []int
	
	for i := 0; i < amount; i++ {
		cable := models.Cable{
			Connector1: connector1,
			Connector2: connector2,
			Type:       cableType,
			Length:     length,
			MM2:        mm2,
		}
		
		if err := h.cableRepo.Create(&cable); err != nil {
			log.Printf("❌ Error creating cable %d of %d: %v", i+1, amount, err)
			h.renderCableFormWithError(c, fmt.Sprintf("Error creating cable %d of %d: %v", i+1, amount, err), &cable)
			return
		}
		
		createdCables = append(createdCables, cable)
		createdIDs = append(createdIDs, cable.CableID)
	}
	
	log.Printf("✅ Successfully created %d cables with IDs: %v", amount, createdIDs)
	c.Redirect(http.StatusFound, "/cables")
}

func (h *CableHandler) renderCableFormWithError(c *gin.Context, errorMsg string, cable *models.Cable) {
	user, _ := GetCurrentUser(c)
	
	types, _ := h.cableRepo.GetAllCableTypes()
	connectors, _ := h.cableRepo.GetAllCableConnectors()
	
	if cable == nil {
		cable = &models.Cable{}
	}
	
	c.HTML(http.StatusInternalServerError, "cable_form.html", gin.H{
		"title":      "New Cable",
		"cable":      cable,
		"types":      types,
		"connectors": connectors,
		"error":      errorMsg,
		"user":       user,
	})
}

// API handlers
func (h *CableHandler) ListCablesAPI(c *gin.Context) {
	params := &models.FilterParams{}
	if err := c.ShouldBindQuery(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cables, err := h.cableRepo.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cables": cables})
}

func (h *CableHandler) GetCableAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cable ID"})
		return
	}
	
	cable, err := h.cableRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cable not found"})
		return
	}

	log.Printf("🐛 DEBUG GetCableAPI: Cable ID=%d, Type=%d, Connector1=%d, Connector2=%d", cable.CableID, cable.Type, cable.Connector1, cable.Connector2)
	log.Printf("🐛 DEBUG GetCableAPI: TypeInfo=%+v", cable.TypeInfo)
	log.Printf("🐛 DEBUG GetCableAPI: Connector1Info=%+v", cable.Connector1Info)
	log.Printf("🐛 DEBUG GetCableAPI: Connector2Info=%+v", cable.Connector2Info)

	c.JSON(http.StatusOK, gin.H{"cable": cable})
}

func (h *CableHandler) CreateCableAPI(c *gin.Context) {
	var cable models.Cable
	if err := c.ShouldBindJSON(&cable); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cable data"})
		return
	}

	if err := h.cableRepo.Create(&cable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cable"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"cable": cable})
}

func (h *CableHandler) UpdateCableAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cable ID"})
		return
	}

	var cable models.Cable
	if err := c.ShouldBindJSON(&cable); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cable data"})
		return
	}

	cable.CableID = id
	if err := h.cableRepo.Update(&cable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cable": cable})
}

func (h *CableHandler) DeleteCableAPI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cable ID"})
		return
	}

	if err := h.cableRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cable deleted successfully"})
}

func (h *CableHandler) GetCableTypesAPI(c *gin.Context) {
	types, err := h.cableRepo.GetAllCableTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cable types"})
		return
	}

	log.Printf("🐛 DEBUG GetCableTypesAPI: Found %d types", len(types))
	for i, t := range types {
		log.Printf("🐛 DEBUG GetCableTypesAPI: Type[%d] ID=%d, Name=%s", i, t.CableTypesID, t.Name)
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

func (h *CableHandler) GetCableConnectorsAPI(c *gin.Context) {
	connectors, err := h.cableRepo.GetAllCableConnectors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cable connectors"})
		return
	}

	log.Printf("🐛 DEBUG GetCableConnectorsAPI: Found %d connectors", len(connectors))
	for i, conn := range connectors {
		log.Printf("🐛 DEBUG GetCableConnectorsAPI: Connector[%d] ID=%d, Name=%s", i, conn.CableConnectorsID, conn.Name)
	}

	c.JSON(http.StatusOK, gin.H{"connectors": connectors})
}