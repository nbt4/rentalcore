package models

import "time"

type Venue struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	Name        string  `json:"name" gorm:"column:name;not null"`
	Street      *string `json:"street" gorm:"column:street"`
	HouseNumber *string `json:"house_number" gorm:"column:house_number"`
	ZIP         *string `json:"zip" gorm:"column:zip"`
	City        *string `json:"city" gorm:"column:city"`
	ContactName *string `json:"contact_name" gorm:"column:contact_name"`
	Phone       *string `json:"phone" gorm:"column:phone"`
	Email       *string `json:"email" gorm:"column:email"`
	Notes       *string `json:"notes" gorm:"column:notes"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Venue) TableName() string {
	return "venues"
}
