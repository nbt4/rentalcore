package models

import (
	"database/sql"
	"time"
)

// ProductPackage mirrors the shared WarehouseCore product_packages table.
// Live DB PK column is "id" (not "package_id").
type ProductPackage struct {
	PackageID   int             `gorm:"primaryKey;column:id" json:"package_id"`
	Name        string          `gorm:"column:name" json:"name"`
	PackageCode string          `gorm:"column:package_code" json:"package_code"`
	Code        string          `gorm:"column:code" json:"code"`
	Description sql.NullString  `gorm:"column:description" json:"description"`
	Price       sql.NullFloat64 `gorm:"column:price" json:"price"`
	IsActive    bool            `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt   time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at" json:"updated_at"`
}

func (ProductPackage) TableName() string {
	return "product_packages"
}

// ProductPackageItem maps a product to a package with a fixed quantity.
// package_id here is a FK to product_packages(id).
type ProductPackageItem struct {
	PackageItemID int `gorm:"primaryKey;column:package_item_id" json:"package_item_id"`
	PackageID     int `gorm:"column:package_id" json:"package_id"`
	ProductID     int `gorm:"column:product_id" json:"product_id"`
	Quantity      int `gorm:"column:quantity" json:"quantity"`
}

func (ProductPackageItem) TableName() string {
	return "product_package_items"
}
