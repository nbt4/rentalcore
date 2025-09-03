package repository

import (
	"log"
	"go-barcode-webapp/internal/models"
)

type ProductRepository struct {
	db *Database
}

func NewProductRepository(db *Database) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetDB returns the database connection for direct queries
func (r *ProductRepository) GetDB() *Database {
	return r.db
}

func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) GetByID(id uint) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Category").
		Preload("Subcategory").
		Preload("Subbiercategory").
		Preload("Brand").
		First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(id uint) error {
	return r.db.Delete(&models.Product{}, id).Error
}

func (r *ProductRepository) List(params *models.FilterParams) ([]models.Product, error) {
	var products []models.Product

	query := r.db.Model(&models.Product{}).
		Preload("Category").
		Preload("Subcategory").
		Preload("Subbiercategory").
		Preload("Brand")

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	query = query.Order("name ASC")

	err := query.Find(&products).Error
	return products, err
}

func (r *ProductRepository) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Order("name ASC").Find(&categories).Error
	if err != nil {
		log.Printf("❌ GetAllCategories error: %v", err)
		return nil, err
	}
	log.Printf("🔧 GetAllCategories: Found %d categories in database", len(categories))
	for _, cat := range categories {
		log.Printf("🔧 Category from DB: %s (ID: %d)", cat.Name, cat.CategoryID)
	}
	return categories, err
}


func (r *ProductRepository) GetDevicesBySubbiercategory(subbiercategoryID string) ([]models.DeviceWithJobInfo, error) {
	var devices []models.Device
	
	err := r.db.Model(&models.Device{}).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Joins("LEFT JOIN products ON products.productID = devices.productID").
		Where("products.subbiercategoryID = ?", subbiercategoryID).
		Order("devices.serialnumber ASC").
		Find(&devices).Error
	
	if err != nil {
		return nil, err
	}
	
	// Convert to DeviceWithJobInfo format
	var result []models.DeviceWithJobInfo
	for _, device := range devices {
		// Check if device is assigned to any job
		var jobDevice models.JobDevice
		err := r.db.Where("deviceID = ?", device.DeviceID).First(&jobDevice).Error
		var jobID *uint
		isAssigned := false
		if err == nil {
			jobID = &jobDevice.JobID
			isAssigned = true
		}
		
		result = append(result, models.DeviceWithJobInfo{
			Device:     device,
			JobID:      jobID,
			IsAssigned: isAssigned,
		})
	}
	
	return result, nil
}

func (r *ProductRepository) GetDevicesBySubcategory(subcategoryID string) ([]models.DeviceWithJobInfo, error) {
	var devices []models.Device
	
	err := r.db.Model(&models.Device{}).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Joins("LEFT JOIN products ON products.productID = devices.productID").
		Where("products.subcategoryID = ? AND (products.subbiercategoryID IS NULL OR products.subbiercategoryID = '' OR products.subbiercategoryID = '0')", subcategoryID).
		Order("devices.serialnumber ASC").
		Find(&devices).Error
	
	if err != nil {
		return nil, err
	}
	
	// Convert to DeviceWithJobInfo format
	var result []models.DeviceWithJobInfo
	for _, device := range devices {
		// Check if device is assigned to any job
		var jobDevice models.JobDevice
		err := r.db.Where("deviceID = ?", device.DeviceID).First(&jobDevice).Error
		var jobID *uint
		isAssigned := false
		if err == nil {
			jobID = &jobDevice.JobID
			isAssigned = true
		}
		
		result = append(result, models.DeviceWithJobInfo{
			Device:     device,
			JobID:      jobID,
			IsAssigned: isAssigned,
		})
	}
	
	return result, nil
}

func (r *ProductRepository) GetDevicesByCategory(categoryID uint) ([]models.DeviceWithJobInfo, error) {
	log.Printf("🔍 GetDevicesByCategory: Searching for devices in category %d", categoryID)
	var devices []models.Device
	
	err := r.db.Model(&models.Device{}).
		Preload("Product").
		Preload("Product.Category").
		Preload("Product.Subcategory").
		Preload("Product.Subbiercategory").
		Joins("LEFT JOIN products ON products.productID = devices.productID").
		Where("products.categoryID = ?", categoryID).
		Order("devices.serialnumber ASC").
		Find(&devices).Error
	
	if err != nil {
		log.Printf("❌ GetDevicesByCategory: Database error for category %d: %v", categoryID, err)
		return nil, err
	}
	
	log.Printf("🔍 GetDevicesByCategory: Found %d devices for category %d", len(devices), categoryID)
	
	// Convert to DeviceWithJobInfo format
	var result []models.DeviceWithJobInfo
	for _, device := range devices {
		// Check if device is assigned to any job
		var jobDevice models.JobDevice
		err := r.db.Where("deviceID = ?", device.DeviceID).First(&jobDevice).Error
		var jobID *uint
		isAssigned := false
		if err == nil {
			jobID = &jobDevice.JobID
			isAssigned = true
		}
		
		result = append(result, models.DeviceWithJobInfo{
			Device:     device,
			JobID:      jobID,
			IsAssigned: isAssigned,
		})
	}
	
	return result, nil
}

// GetSubcategoriesByCategory gets all subcategories for a given category
func (r *ProductRepository) GetSubcategoriesByCategory(categoryID uint, subcategories *[]models.Subcategory) error {
	return r.db.Where("categoryID = ?", categoryID).Order("name ASC").Find(subcategories).Error
}

// GetSubbiercategoriesBySubcategory gets all subbiercategories for a given subcategory
func (r *ProductRepository) GetSubbiercategoriesBySubcategory(subcategoryID string, subbiercategories *[]models.Subbiercategory) error {
	return r.db.Where("subcategoryID = ?", subcategoryID).Order("name ASC").Find(subbiercategories).Error
}

// GetAllSubcategories gets all subcategories
func (r *ProductRepository) GetAllSubcategories(subcategories *[]models.Subcategory) error {
	return r.db.Order("name ASC").Find(subcategories).Error
}

// GetAllSubbiercategories gets all subbiercategories
func (r *ProductRepository) GetAllSubbiercategories(subbiercategories *[]models.Subbiercategory) error {
	return r.db.Order("name ASC").Find(subbiercategories).Error
}

// GetAllBrands gets all brands
func (r *ProductRepository) GetAllBrands(brands *[]models.Brand) error {
	return r.db.Order("name ASC").Find(brands).Error
}

// GetAllManufacturers gets all manufacturers
func (r *ProductRepository) GetAllManufacturers(manufacturers *[]models.Manufacturer) error {
	return r.db.Order("name ASC").Find(manufacturers).Error
}