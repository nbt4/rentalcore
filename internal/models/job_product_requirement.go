package models

import (
	"time"

	"gorm.io/gorm"
)

// JobProductRequirement records the required quantity of a product for a job.
// This is Stage 1 of the two-stage device availability model:
// the planner says "I need 3× GLXD4 for this job" without touching devices.
type JobProductRequirement struct {
	ID               uint           `gorm:"primaryKey;column:id" json:"id"`
	JobID            uint           `gorm:"column:job_id;not null;index" json:"job_id"`
	ProductID        uint           `gorm:"column:product_id;not null" json:"product_id"`
	Quantity         int            `gorm:"column:quantity;not null;default:1" json:"quantity"`
	ManualQuantity   int            `gorm:"column:manual_quantity;not null;default:0" json:"manual_quantity"`
	PositionQuantity int            `gorm:"column:position_quantity;not null;default:0" json:"position_quantity"`
	CreatedAt        time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at"`
	Product          *Product       `gorm:"foreignKey:ProductID;references:ProductID" json:"product,omitempty"`
}

func (JobProductRequirement) TableName() string {
	return "job_product_requirements"
}
