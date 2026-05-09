package models

import "time"

type JobPosition struct {
	PositionID         uint      `gorm:"primaryKey;column:position_id" json:"position_id"`
	JobID              uint      `gorm:"column:job_id;not null;index" json:"job_id"`
	PositionType       string    `gorm:"column:position_type;not null;default:product" json:"position_type"`
	ProductID          *uint     `gorm:"column:product_id" json:"product_id"`
	ServiceItemID      *uint     `gorm:"column:service_item_id" json:"service_item_id"`
	RentalEquipmentID  *uint     `gorm:"column:rental_equipment_id" json:"rental_equipment_id"`
	Description        string    `gorm:"column:description;not null;default:''" json:"description"`
	Quantity           float64   `gorm:"column:quantity;not null;default:1" json:"quantity"`
	Unit               string    `gorm:"column:unit;not null;default:Stück" json:"unit"`
	UnitPrice          float64   `gorm:"column:unit_price;not null;default:0" json:"unit_price"`
	FollowDayFactor    float64   `gorm:"column:follow_day_factor;not null;default:0.50" json:"follow_day_factor"`
	DiscountPercent    float64   `gorm:"column:discount_percent;not null;default:0" json:"discount_percent"`
	DiscountAmount     float64   `gorm:"column:discount_amount;not null;default:0" json:"discount_amount"`
	TaxRate            float64   `gorm:"column:tax_rate;not null;default:19.00" json:"tax_rate"`
	SortOrder          int       `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt          time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Product          *Product          `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
	ServiceItem      *ServiceItem      `gorm:"foreignKey:ServiceItemID;references:ID" json:"service_item,omitempty"`
	RentalEquipment  *RentalEquipment  `gorm:"foreignKey:RentalEquipmentID;references:EquipmentID" json:"rental_equipment,omitempty"`
	Devices          []JobPositionDevice `gorm:"foreignKey:PositionID;references:PositionID" json:"devices,omitempty"`
}

func (JobPosition) TableName() string { return "job_positions" }

type JobPositionDevice struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	PositionID uint      `gorm:"column:position_id;not null;index" json:"position_id"`
	DeviceID   string    `gorm:"column:device_id;not null" json:"device_id"`
	ScannedAt  time.Time `gorm:"column:scanned_at;default:CURRENT_TIMESTAMP" json:"scanned_at"`
	ScannedBy  string    `gorm:"column:scanned_by;default:''" json:"scanned_by"`
}

func (JobPositionDevice) TableName() string { return "job_position_devices" }

type ServiceItem struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	Name         string    `gorm:"column:name;not null" json:"name"`
	Description  string    `gorm:"column:description" json:"description"`
	DefaultPrice float64   `gorm:"column:default_price;default:0" json:"default_price"`
	Category     string    `gorm:"column:category" json:"category"`
	Unit         string    `gorm:"column:unit;default:pauschal" json:"unit"`
	IsActive     bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ServiceItem) TableName() string { return "service_items" }
