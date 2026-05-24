package models

import "time"

type Skill struct {
	ID          uint      `json:"id"          gorm:"primaryKey"`
	Name        string    `json:"name"        gorm:"not null;uniqueIndex;size:100"`
	Category    string    `json:"category"    gorm:"not null;default:''"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Skill) TableName() string { return "skills" }

type Employee struct {
	ID          uint       `json:"id"           gorm:"primaryKey"`
	FirstName   string     `json:"first_name"   gorm:"not null;size:100"`
	LastName    string     `json:"last_name"    gorm:"not null;size:100"`
	Email       *string    `json:"email"        gorm:"uniqueIndex;size:255"`
	Phone       *string    `json:"phone"        gorm:"size:50"`
	Mobile      *string    `json:"mobile"       gorm:"size:50"`
	Street      *string    `json:"street"       gorm:"size:255"`
	HouseNumber *string    `json:"house_number" gorm:"size:20"`
	ZIP         *string    `json:"zip"          gorm:"column:zip;size:20"`
	City        *string    `json:"city"         gorm:"size:100"`
	Country     *string    `json:"country"      gorm:"default:'Deutschland'"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	IBAN        *string    `json:"iban"         gorm:"column:iban;size:50"`
	Notes       *string    `json:"notes"`
	IsActive    bool       `json:"is_active"    gorm:"default:true"`
	Skills      []Skill    `json:"skills"       gorm:"many2many:employee_skills;"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Employee) TableName() string { return "employees" }

func (e Employee) DisplayName() string {
	return e.FirstName + " " + e.LastName
}

type JobEmployee struct {
	JobID        uint      `json:"job_id"        gorm:"primaryKey"`
	EmployeeID   uint      `json:"employee_id"   gorm:"primaryKey"`
	Role         *string   `json:"role"          gorm:"size:100"`
	M365EventID  *string   `json:"-"             gorm:"column:m365_event_id;size:512"`
	CreatedAt    time.Time `json:"created_at"`
	Employee     Employee  `json:"employee"      gorm:"foreignKey:EmployeeID"`
}

func (JobEmployee) TableName() string { return "job_employees" }
