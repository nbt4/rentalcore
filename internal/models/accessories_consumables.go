package models

import "time"

// CountType represents a measurement unit for accessories and consumables
type CountType struct {
	CountTypeID  uint      `json:"count_type_id" gorm:"primaryKey;column:count_type_id"`
	Name         string    `json:"name" gorm:"not null;column:name"`
	Abbreviation string    `json:"abbreviation" gorm:"not null;column:abbreviation"`
	IsActive     bool      `json:"is_active" gorm:"column:is_active;default:1"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (CountType) TableName() string {
	return "count_types"
}

// ProductAccessory links a product to its available accessories
type ProductAccessory struct {
	ProductID          uint      `json:"product_id" gorm:"primaryKey;column:product_id"`
	AccessoryProductID uint      `json:"accessory_product_id" gorm:"primaryKey;column:accessory_product_id"`
	IsOptional         bool      `json:"is_optional" gorm:"column:is_optional;default:1"`
	DefaultQuantity    int       `json:"default_quantity" gorm:"column:default_quantity;default:1"`
	SortOrder          *int      `json:"sort_order" gorm:"column:sort_order"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`

	// Relations
	Product          *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	AccessoryProduct *Product `json:"accessory_product,omitempty" gorm:"foreignKey:AccessoryProductID;references:ProductID"`
}

func (ProductAccessory) TableName() string {
	return "product_accessories"
}

// ProductConsumable links a product to its available consumables
type ProductConsumable struct {
	ProductID           uint      `json:"product_id" gorm:"primaryKey;column:product_id"`
	ConsumableProductID uint      `json:"consumable_product_id" gorm:"primaryKey;column:consumable_product_id"`
	DefaultQuantity     float64   `json:"default_quantity" gorm:"column:default_quantity;type:decimal(10,3);default:1.000"`
	SortOrder           *int      `json:"sort_order" gorm:"column:sort_order"`
	CreatedAt           time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`

	// Relations
	Product           *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	ConsumableProduct *Product `json:"consumable_product,omitempty" gorm:"foreignKey:ConsumableProductID;references:ProductID"`
}

func (ProductConsumable) TableName() string {
	return "product_consumables"
}

// ProductDependency represents the new unified dependency table from WarehouseCore
type ProductDependency struct {
	ID                  int       `json:"id" gorm:"primaryKey;column:id;autoIncrement"`
	ProductID           int       `json:"product_id" gorm:"not null;column:product_id;index"`
	DependencyProductID int       `json:"dependency_product_id" gorm:"not null;column:dependency_product_id;index"`
	IsOptional          bool      `json:"is_optional" gorm:"column:is_optional;default:1"`
	DefaultQuantity     float64   `json:"default_quantity" gorm:"column:default_quantity;type:decimal(10,2);default:1.00"`
	Notes               *string   `json:"notes" gorm:"column:notes;type:varchar(500)"`
	CreatedAt           time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`

	// Relations
	Product           *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	DependencyProduct *Product `json:"dependency_product,omitempty" gorm:"foreignKey:DependencyProductID;references:ProductID"`
}

func (ProductDependency) TableName() string {
	return "product_dependencies"
}

// ProductDependencyView represents a denormalized view of product dependencies with product details
type ProductDependencyView struct {
	ID                  int      `json:"id" gorm:"column:id"`
	ProductID           int      `json:"product_id" gorm:"column:product_id"`
	DependencyProductID int      `json:"dependency_product_id" gorm:"column:dependency_product_id"`
	DependencyName      string   `json:"dependency_name" gorm:"column:dependency_name"`
	IsAccessory         bool     `json:"is_accessory" gorm:"column:is_accessory"`
	IsConsumable        bool     `json:"is_consumable" gorm:"column:is_consumable"`
	GenericBarcode      *string  `json:"generic_barcode" gorm:"column:generic_barcode"`
	CountTypeAbbr       *string  `json:"count_type_abbr" gorm:"column:count_type_abbr"`
	StockQuantity       *float64 `json:"stock_quantity" gorm:"column:stock_quantity"`
	IsOptional          bool     `json:"is_optional" gorm:"column:is_optional"`
	DefaultQuantity     float64  `json:"default_quantity" gorm:"column:default_quantity"`
	Notes               *string  `json:"notes" gorm:"column:notes"`
}

// JobAccessory tracks accessories assigned to a job
type JobAccessory struct {
	JobAccessoryID     uint64    `json:"job_accessory_id" gorm:"primaryKey;column:job_accessory_id;autoIncrement"`
	JobID              uint      `json:"job_id" gorm:"not null;column:job_id;index"`
	ParentDeviceID     *string   `json:"parent_device_id" gorm:"column:parent_device_id;index"`
	AccessoryProductID uint      `json:"accessory_product_id" gorm:"not null;column:accessory_product_id;index"`
	QuantityAssigned   int       `json:"quantity_assigned" gorm:"column:quantity_assigned;default:1"`
	QuantityScannedOut int       `json:"quantity_scanned_out" gorm:"column:quantity_scanned_out;default:0"`
	QuantityScannedIn  int       `json:"quantity_scanned_in" gorm:"column:quantity_scanned_in;default:0"`
	PricePerUnit       *float64  `json:"price_per_unit" gorm:"column:price_per_unit;type:decimal(10,2)"`
	Notes              *string   `json:"notes" gorm:"column:notes;type:text"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`

	// Relations
	Job              *Job     `json:"job,omitempty" gorm:"foreignKey:JobID;references:JobID"`
	ParentDevice     *Device  `json:"parent_device,omitempty" gorm:"foreignKey:ParentDeviceID;references:DeviceID"`
	AccessoryProduct *Product `json:"accessory_product,omitempty" gorm:"foreignKey:AccessoryProductID;references:ProductID"`
}

func (JobAccessory) TableName() string {
	return "job_accessories"
}

// JobConsumable tracks consumables assigned to a job
type JobConsumable struct {
	JobConsumableID     uint64    `json:"job_consumable_id" gorm:"primaryKey;column:job_consumable_id;autoIncrement"`
	JobID               uint      `json:"job_id" gorm:"not null;column:job_id;index"`
	ParentDeviceID      *string   `json:"parent_device_id" gorm:"column:parent_device_id;index"`
	ConsumableProductID uint      `json:"consumable_product_id" gorm:"not null;column:consumable_product_id;index"`
	QuantityAssigned    float64   `json:"quantity_assigned" gorm:"column:quantity_assigned;type:decimal(10,3);default:1.000"`
	QuantityScannedOut  float64   `json:"quantity_scanned_out" gorm:"column:quantity_scanned_out;type:decimal(10,3);default:0.000"`
	QuantityScannedIn   float64   `json:"quantity_scanned_in" gorm:"column:quantity_scanned_in;type:decimal(10,3);default:0.000"`
	PricePerUnit        *float64  `json:"price_per_unit" gorm:"column:price_per_unit;type:decimal(10,2)"`
	Notes               *string   `json:"notes" gorm:"column:notes;type:text"`
	CreatedAt           time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`

	// Relations
	Job               *Job     `json:"job,omitempty" gorm:"foreignKey:JobID;references:JobID"`
	ParentDevice      *Device  `json:"parent_device,omitempty" gorm:"foreignKey:ParentDeviceID;references:DeviceID"`
	ConsumableProduct *Product `json:"consumable_product,omitempty" gorm:"foreignKey:ConsumableProductID;references:ProductID"`
}

func (JobConsumable) TableName() string {
	return "job_consumables"
}

// View Models - These represent the database views for easier querying

// ProductAccessoryView represents the vw_product_accessories view
type ProductAccessoryView struct {
	ProductID          uint     `json:"product_id" gorm:"column:product_id"`
	ProductName        string   `json:"product_name" gorm:"column:product_name"`
	AccessoryProductID uint     `json:"accessory_product_id" gorm:"column:accessory_product_id"`
	AccessoryName      string   `json:"accessory_name" gorm:"column:accessory_name"`
	AccessoryStock     *float64 `json:"accessory_stock" gorm:"column:accessory_stock"`
	AccessoryPrice     *float64 `json:"accessory_price" gorm:"column:accessory_price"`
	CountType          *string  `json:"count_type" gorm:"column:count_type"`
	CountTypeAbbr      *string  `json:"count_type_abbr" gorm:"column:count_type_abbr"`
	IsOptional         bool     `json:"is_optional" gorm:"column:is_optional"`
	DefaultQuantity    int      `json:"default_quantity" gorm:"column:default_quantity"`
	SortOrder          *int     `json:"sort_order" gorm:"column:sort_order"`
	GenericBarcode     *string  `json:"generic_barcode" gorm:"column:generic_barcode"`
}

func (ProductAccessoryView) TableName() string {
	return "vw_product_accessories"
}

// ProductConsumableView represents the vw_product_consumables view
type ProductConsumableView struct {
	ProductID           uint     `json:"product_id" gorm:"column:product_id"`
	ProductName         string   `json:"product_name" gorm:"column:product_name"`
	ConsumableProductID uint     `json:"consumable_product_id" gorm:"column:consumable_product_id"`
	ConsumableName      string   `json:"consumable_name" gorm:"column:consumable_name"`
	ConsumableStock     *float64 `json:"consumable_stock" gorm:"column:consumable_stock"`
	ConsumablePrice     *float64 `json:"consumable_price" gorm:"column:consumable_price"`
	CountType           *string  `json:"count_type" gorm:"column:count_type"`
	CountTypeAbbr       *string  `json:"count_type_abbr" gorm:"column:count_type_abbr"`
	DefaultQuantity     float64  `json:"default_quantity" gorm:"column:default_quantity"`
	SortOrder           *int     `json:"sort_order" gorm:"column:sort_order"`
	GenericBarcode      *string  `json:"generic_barcode" gorm:"column:generic_barcode"`
}

func (ProductConsumableView) TableName() string {
	return "vw_product_consumables"
}

// LowStockAlert represents the vw_low_stock_alert view
type LowStockAlert struct {
	ProductID        uint     `json:"productID" gorm:"column:productID"`
	Name             string   `json:"name" gorm:"column:name"`
	StockQuantity    *float64 `json:"stock_quantity" gorm:"column:stock_quantity"`
	MinStockLevel    *float64 `json:"min_stock_level" gorm:"column:min_stock_level"`
	QuantityBelowMin *float64 `json:"quantity_below_min" gorm:"column:quantity_below_min"`
	CountType        *string  `json:"count_type" gorm:"column:count_type"`
	CountTypeAbbr    *string  `json:"count_type_abbr" gorm:"column:count_type_abbr"`
	GenericBarcode   *string  `json:"generic_barcode" gorm:"column:generic_barcode"`
	ItemType         string   `json:"item_type" gorm:"column:item_type"`
}

func (LowStockAlert) TableName() string {
	return "vw_low_stock_alert"
}
