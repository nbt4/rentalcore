package models

import (
	"database/sql"
	"time"
)

// JobPackage represents a package assigned to a job as a single line item
type JobPackage struct {
	JobPackageID uint            `gorm:"primaryKey;column:job_package_id" json:"job_package_id"`
	JobID        int             `gorm:"column:job_id;not null" json:"job_id"`
	PackageID    int             `gorm:"column:package_id;not null" json:"package_id"`
	Quantity     uint            `gorm:"column:quantity;not null;default:1" json:"quantity"`
	CustomPrice  sql.NullFloat64 `gorm:"column:custom_price" json:"custom_price"`
	AddedAt      time.Time       `gorm:"column:added_at;not null;default:CURRENT_TIMESTAMP" json:"added_at"`
	AddedBy      *uint           `gorm:"column:added_by" json:"added_by"`
	Notes        sql.NullString  `gorm:"column:notes" json:"notes"`

	// Relationships
	Job          *Job                    `gorm:"foreignKey:JobID;references:JobID" json:"job,omitempty"`
	Package      *ProductPackage         `gorm:"foreignKey:PackageID;references:PackageID" json:"package,omitempty"`
	AddedByUser  *User                   `gorm:"foreignKey:AddedBy;references:UserID" json:"added_by_user,omitempty"`
	Reservations []JobPackageReservation `gorm:"foreignKey:JobPackageID" json:"reservations,omitempty"`
}

// TableName overrides the default table name
func (JobPackage) TableName() string {
	return "job_packages"
}

// JobPackageReservation tracks device reservations for packages assigned to jobs
type JobPackageReservation struct {
	ReservationID     uint         `gorm:"primaryKey;column:reservation_id" json:"reservation_id"`
	JobPackageID      uint         `gorm:"column:job_package_id;not null" json:"job_package_id"`
	DeviceID          string       `gorm:"column:device_id;not null" json:"device_id"`
	Quantity          uint         `gorm:"column:quantity;not null;default:1" json:"quantity"`
	ReservationStatus string       `gorm:"column:reservation_status;not null;default:'reserved'" json:"reservation_status"`
	ReservedAt        time.Time    `gorm:"column:reserved_at;not null;default:CURRENT_TIMESTAMP" json:"reserved_at"`
	AssignedAt        sql.NullTime `gorm:"column:assigned_at" json:"assigned_at"`
	ReleasedAt        sql.NullTime `gorm:"column:released_at" json:"released_at"`

	// Relationships
	JobPackage *JobPackage `gorm:"foreignKey:JobPackageID;references:JobPackageID" json:"job_package,omitempty"`
	Device     *Device     `gorm:"foreignKey:DeviceID;references:DeviceID" json:"device,omitempty"`
}

// TableName overrides the default table name
func (JobPackageReservation) TableName() string {
	return "job_package_reservations"
}

// JobPackageWithDetails extends JobPackage with computed fields for display
type JobPackageWithDetails struct {
	JobPackage
	PackageName        string  `json:"package_name"`
	PackageDescription string  `json:"package_description"`
	PackagePrice       float64 `json:"package_price"`
	EffectivePrice     float64 `json:"effective_price"`
	TotalPrice         float64 `json:"total_price"`
	DeviceCount        int     `json:"device_count"`
	ReservedDevices    int     `json:"reserved_devices"`
	AvailabilityStatus string  `json:"availability_status"`
}
