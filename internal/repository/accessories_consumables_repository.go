package repository

import (
	"fmt"

	"go-barcode-webapp/internal/models"
	"gorm.io/gorm"

	"go-barcode-webapp/internal/logger"
)

type AccessoriesConsumablesRepository struct {
	db *Database
}

func NewAccessoriesConsumablesRepository(db *Database) *AccessoriesConsumablesRepository {
	return &AccessoriesConsumablesRepository{db: db}
}

// ============================================================================
// Count Types
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetAllCountTypes() ([]models.CountType, error) {
	var countTypes []models.CountType
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&countTypes).Error
	return countTypes, err
}

func (r *AccessoriesConsumablesRepository) GetCountTypeByID(id uint) (*models.CountType, error) {
	var countType models.CountType
	err := r.db.First(&countType, id).Error
	if err != nil {
		return nil, err
	}
	return &countType, nil
}

// ============================================================================
// Product Dependencies (WarehouseCore integration)
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetProductDependencies(productID uint) ([]models.ProductDependencyView, error) {
	var dependencies []models.ProductDependencyView
	query := `
		SELECT
			pd.id,
			pd.product_id,
			pd.dependency_product_id,
			p.name as dependency_name,
			COALESCE(p.is_accessory, false) as is_accessory,
			COALESCE(p.is_consumable, false) as is_consumable,
			p.generic_barcode,
			ct.abbreviation as count_type_abbr,
			p.stock_quantity,
			pd.is_optional,
			pd.default_quantity,
			pd.notes
		FROM product_dependencies pd
		JOIN products p ON pd.dependency_product_id = p.productid
		LEFT JOIN count_types ct ON p.count_type_id = ct.count_type_id
		WHERE pd.product_id = ?
		ORDER BY pd.is_optional ASC, pd.created_at DESC
	`
	err := r.db.Raw(query, productID).Scan(&dependencies).Error
	return dependencies, err
}

// ============================================================================
// Product Accessories
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetProductAccessories(productID uint) ([]models.ProductAccessoryView, error) {
	var accessories []models.ProductAccessoryView
	err := r.db.Where("product_id = ?", productID).Order("sort_order ASC, accessory_name ASC").Find(&accessories).Error
	return accessories, err
}

func (r *AccessoriesConsumablesRepository) AddProductAccessory(pa *models.ProductAccessory) error {
	return r.db.Create(pa).Error
}

func (r *AccessoriesConsumablesRepository) RemoveProductAccessory(productID, accessoryProductID uint) error {
	return r.db.Where("product_id = ? AND accessory_product_id = ?", productID, accessoryProductID).
		Delete(&models.ProductAccessory{}).Error
}

func (r *AccessoriesConsumablesRepository) UpdateProductAccessory(pa *models.ProductAccessory) error {
	return r.db.Save(pa).Error
}

// Get list of all accessory products (is_accessory = true)
func (r *AccessoriesConsumablesRepository) GetAccessoryProducts() ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("is_accessory = ?", true).
		Preload("CountType").
		Order("name ASC").
		Find(&products).Error
	return products, err
}

// ============================================================================
// Product Consumables
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetProductConsumables(productID uint) ([]models.ProductConsumableView, error) {
	var consumables []models.ProductConsumableView
	err := r.db.Where("product_id = ?", productID).Order("sort_order ASC, consumable_name ASC").Find(&consumables).Error
	return consumables, err
}

func (r *AccessoriesConsumablesRepository) AddProductConsumable(pc *models.ProductConsumable) error {
	return r.db.Create(pc).Error
}

func (r *AccessoriesConsumablesRepository) RemoveProductConsumable(productID, consumableProductID uint) error {
	return r.db.Where("product_id = ? AND consumable_product_id = ?", productID, consumableProductID).
		Delete(&models.ProductConsumable{}).Error
}

func (r *AccessoriesConsumablesRepository) UpdateProductConsumable(pc *models.ProductConsumable) error {
	return r.db.Save(pc).Error
}

// Get list of all consumable products (is_consumable = true)
func (r *AccessoriesConsumablesRepository) GetConsumableProducts() ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("is_consumable = ?", true).
		Preload("CountType").
		Order("name ASC").
		Find(&products).Error
	return products, err
}

// ============================================================================
// Job Accessories
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetJobAccessories(jobID uint) ([]models.JobAccessory, error) {
	var accessories []models.JobAccessory
	err := r.db.Where("job_id = ?", jobID).
		Preload("AccessoryProduct").
		Preload("AccessoryProduct.CountType").
		Preload("ParentDevice").
		Order("job_accessory_id ASC").
		Find(&accessories).Error
	return accessories, err
}

func (r *AccessoriesConsumablesRepository) GetJobAccessoryByID(id uint64) (*models.JobAccessory, error) {
	var accessory models.JobAccessory
	err := r.db.Preload("AccessoryProduct").
		Preload("AccessoryProduct.CountType").
		Preload("ParentDevice").
		First(&accessory, id).Error
	if err != nil {
		return nil, err
	}
	return &accessory, nil
}

func (r *AccessoriesConsumablesRepository) CreateJobAccessory(ja *models.JobAccessory) error {
	return r.db.Create(ja).Error
}

func (r *AccessoriesConsumablesRepository) UpdateJobAccessory(ja *models.JobAccessory) error {
	return r.db.Save(ja).Error
}

func (r *AccessoriesConsumablesRepository) DeleteJobAccessory(id uint64) error {
	return r.db.Delete(&models.JobAccessory{}, id).Error
}

// Scan accessories out for a job
func (r *AccessoriesConsumablesRepository) ScanAccessoryOut(jobAccessoryID uint64, quantity int, userID *uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var ja models.JobAccessory
		if err := tx.First(&ja, jobAccessoryID).Error; err != nil {
			return err
		}

		// Update scanned out quantity
		ja.QuantityScannedOut += quantity

		// Validate we're not scanning more than assigned
		if ja.QuantityScannedOut > ja.QuantityAssigned {
			return fmt.Errorf("cannot scan out more than assigned quantity")
		}

		if err := tx.Save(&ja).Error; err != nil {
			return err
		}

		// Decrease stock
		if err := tx.Model(&models.Product{}).
			Where("productID = ?", ja.AccessoryProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity - ?", quantity)).Error; err != nil {
			return err
		}

		// Transaction logging removed with inventory system
		return nil
	})
}

// Scan accessories in from a job
func (r *AccessoriesConsumablesRepository) ScanAccessoryIn(jobAccessoryID uint64, quantity int, userID *uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var ja models.JobAccessory
		if err := tx.First(&ja, jobAccessoryID).Error; err != nil {
			return err
		}

		// Update scanned in quantity
		ja.QuantityScannedIn += quantity

		// Validate we're not scanning in more than scanned out
		if ja.QuantityScannedIn > ja.QuantityScannedOut {
			return fmt.Errorf("cannot scan in more than scanned out quantity")
		}

		if err := tx.Save(&ja).Error; err != nil {
			return err
		}

		// Increase stock
		if err := tx.Model(&models.Product{}).
			Where("productID = ?", ja.AccessoryProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity)).Error; err != nil {
			return err
		}

		// Transaction logging removed with inventory system
		return nil
	})
}

// ============================================================================
// Job Consumables
// ============================================================================

func (r *AccessoriesConsumablesRepository) GetJobConsumables(jobID uint) ([]models.JobConsumable, error) {
	var consumables []models.JobConsumable
	err := r.db.Where("job_id = ?", jobID).
		Preload("ConsumableProduct").
		Preload("ConsumableProduct.CountType").
		Preload("ParentDevice").
		Order("job_consumable_id ASC").
		Find(&consumables).Error
	return consumables, err
}

func (r *AccessoriesConsumablesRepository) GetJobConsumableByID(id uint64) (*models.JobConsumable, error) {
	var consumable models.JobConsumable
	err := r.db.Preload("ConsumableProduct").
		Preload("ConsumableProduct.CountType").
		Preload("ParentDevice").
		First(&consumable, id).Error
	if err != nil {
		return nil, err
	}
	return &consumable, nil
}

func (r *AccessoriesConsumablesRepository) CreateJobConsumable(jc *models.JobConsumable) error {
	return r.db.Create(jc).Error
}

func (r *AccessoriesConsumablesRepository) UpdateJobConsumable(jc *models.JobConsumable) error {
	return r.db.Save(jc).Error
}

func (r *AccessoriesConsumablesRepository) DeleteJobConsumable(id uint64) error {
	return r.db.Delete(&models.JobConsumable{}, id).Error
}

// Scan consumables out for a job
func (r *AccessoriesConsumablesRepository) ScanConsumableOut(jobConsumableID uint64, quantity float64, userID *uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var jc models.JobConsumable
		if err := tx.First(&jc, jobConsumableID).Error; err != nil {
			return err
		}

		// Update scanned out quantity
		jc.QuantityScannedOut += quantity

		// Validate we're not scanning more than assigned
		if jc.QuantityScannedOut > jc.QuantityAssigned {
			return fmt.Errorf("cannot scan out more than assigned quantity")
		}

		if err := tx.Save(&jc).Error; err != nil {
			return err
		}

		// Decrease stock
		if err := tx.Model(&models.Product{}).
			Where("productID = ?", jc.ConsumableProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity - ?", quantity)).Error; err != nil {
			return err
		}

		// Transaction logging removed with inventory system
		return nil
	})
}

// Scan consumables in from a job
func (r *AccessoriesConsumablesRepository) ScanConsumableIn(jobConsumableID uint64, quantity float64, userID *uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var jc models.JobConsumable
		if err := tx.First(&jc, jobConsumableID).Error; err != nil {
			return err
		}

		// Update scanned in quantity
		jc.QuantityScannedIn += quantity

		// Validate we're not scanning in more than scanned out
		if jc.QuantityScannedIn > jc.QuantityScannedOut {
			return fmt.Errorf("cannot scan in more than scanned out quantity")
		}

		if err := tx.Save(&jc).Error; err != nil {
			return err
		}

		// Increase stock
		if err := tx.Model(&models.Product{}).
			Where("productID = ?", jc.ConsumableProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity)).Error; err != nil {
			return err
		}

		// Transaction logging removed with inventory system
		return nil
	})
}


// ============================================================================
// Barcode Operations
// ============================================================================

// GetAccessoryByBarcode finds an accessory product by its generic barcode
func (r *AccessoriesConsumablesRepository) GetAccessoryByBarcode(barcode string) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("generic_barcode = ? AND is_accessory = ?", barcode, true).
		Preload("CountType").
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetConsumableByBarcode finds a consumable product by its generic barcode
func (r *AccessoriesConsumablesRepository) GetConsumableByBarcode(barcode string) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("generic_barcode = ? AND is_consumable = ?", barcode, true).
		Preload("CountType").
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetJobAccessoriesByJobAndProduct finds job accessories by job ID and accessory product ID
func (r *AccessoriesConsumablesRepository) GetJobAccessoriesByJobAndProduct(jobID, accessoryProductID uint) ([]models.JobAccessory, error) {
	var accessories []models.JobAccessory
	err := r.db.Where("job_id = ? AND accessory_product_id = ?", jobID, accessoryProductID).
		Preload("AccessoryProduct").
		Preload("AccessoryProduct.CountType").
		Find(&accessories).Error
	return accessories, err
}

// GetJobConsumablesByJobAndProduct finds job consumables by job ID and consumable product ID
func (r *AccessoriesConsumablesRepository) GetJobConsumablesByJobAndProduct(jobID, consumableProductID uint) ([]models.JobConsumable, error) {
	var consumables []models.JobConsumable
	err := r.db.Where("job_id = ? AND consumable_product_id = ?", jobID, consumableProductID).
		Preload("ConsumableProduct").
		Preload("ConsumableProduct.CountType").
		Find(&consumables).Error
	return consumables, err
}

// ============================================================================
// Helper functions
// ============================================================================

func strPtr(s string) *string {
	return &s
}

// GetDB returns the database connection for direct queries
func (r *AccessoriesConsumablesRepository) GetDB() *Database {
	return r.db
}

func init() {
	logger.LogInfo("📦 Accessories and Consumables Repository initialized")
}
