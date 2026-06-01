package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	CustomerID   uint    `json:"customer_id" gorm:"primaryKey;column:customerid"`
	CompanyName  *string `json:"companyname" gorm:"column:companyname"`
	LastName     *string `json:"lastname" gorm:"column:lastname"`
	FirstName    *string `json:"firstname" gorm:"column:firstname"`
	Street       *string `json:"street" gorm:"column:street"`
	HouseNumber  *string `json:"housenumber" gorm:"column:housenumber"`
	ZIP          *string `json:"ZIP" gorm:"column:zip"`
	City         *string `json:"city" gorm:"column:city"`
	FederalState *string `json:"federalstate" gorm:"column:federalstate"`
	Country      *string `json:"country" gorm:"column:country"`
	PhoneNumber  *string `json:"phonenumber" gorm:"column:phonenumber"`
	Email        *string `json:"email" gorm:"column:email"`
	CustomerType *string `json:"customertype" gorm:"column:customertype"`
	IsCustomer   bool    `json:"is_customer" gorm:"column:is_customer;default:true"`
	IsSupplier   bool    `json:"is_supplier" gorm:"column:is_supplier;default:false"`
	Notes        *string `json:"notes" gorm:"column:notes"`
	// M365 sync fields
	M365ID        *string    `json:"m365_id,omitempty" gorm:"column:m365_id"`
	M365UpdatedAt *time.Time `json:"m365_updated_at,omitempty" gorm:"column:m365_updated_at"`
	GALContactID  *string    `json:"gal_contact_id,omitempty" gorm:"column:gal_contact_id"`
	IsArchived    bool       `json:"is_archived" gorm:"column:is_archived;default:false"`
	ArchivedAt    *time.Time `json:"archived_at,omitempty" gorm:"column:archived_at"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Jobs          []Job      `json:"jobs,omitempty" gorm:"-"`
}

func (Customer) TableName() string {
	return "customers"
}

func (c Customer) GetDisplayName() string {
	if c.CompanyName != nil && *c.CompanyName != "" {
		return *c.CompanyName
	}
	if c.FirstName != nil && c.LastName != nil && *c.FirstName != "" && *c.LastName != "" {
		return *c.FirstName + " " + *c.LastName
	}
	if c.LastName != nil && *c.LastName != "" {
		return *c.LastName
	}
	if c.FirstName != nil && *c.FirstName != "" {
		return *c.FirstName
	}
	return "Unknown Customer"
}

type Status struct {
	StatusID uint   `json:"status_id" gorm:"primaryKey;column:statusid"`
	Status   string `json:"status" gorm:"not null;column:status"`
	Jobs     []Job  `json:"jobs,omitempty" gorm:"-"`
}

func (Status) TableName() string {
	return "status"
}

type Job struct {
	JobID         uint         `json:"jobID" gorm:"primaryKey;column:jobid"`
	JobCode       string       `json:"job_code" gorm:"column:job_code"`
	CustomerID    uint         `json:"customer_id" gorm:"not null;column:customerid"`
	Customer      Customer     `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	StatusID      uint         `json:"status_id" gorm:"not null;column:statusid"`
	Status        Status       `json:"status,omitempty" gorm:"foreignKey:StatusID"`
	JobCategoryID *uint        `json:"jobcategoryID" gorm:"column:jobcategoryid"`
	JobCategory   *JobCategory `json:"job_category,omitempty" gorm:"foreignKey:JobCategoryID"`
	CreatedBy     *uint        `json:"created_by" gorm:"column:created_by;index"`
	CreatedAt     *time.Time   `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP;index"`
	UpdatedBy     *uint        `json:"updated_by" gorm:"column:updated_by;index"`
	UpdatedAt     *time.Time   `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP;index"`

	// Relations
	Creator      *User        `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Description  *string      `json:"description" gorm:"column:description"`
	Discount     float64      `json:"discount" gorm:"column:discount;default:0"`
	DiscountType string       `json:"discount_type" gorm:"column:discount_type;default:amount"`
	Revenue      float64      `json:"revenue" gorm:"column:revenue;default:0"`
	FinalRevenue    *float64     `json:"final_revenue" gorm:"column:final_revenue"`
	StartDate       *time.Time   `json:"startDate" gorm:"column:startdate;type:date"`
	EndDate         *time.Time   `json:"endDate" gorm:"column:enddate;type:date"`
	MultiplyByDays  bool         `json:"multiply_by_days" gorm:"column:multiply_by_days;default:true"`
	M365EventID     *string      `json:"m365_event_id,omitempty" gorm:"column:m365_event_id;size:255"`
	PricesIncludeTax bool        `json:"prices_include_tax" gorm:"column:prices_include_tax;default:false"`
	VenueID      *uint        `json:"venue_id" gorm:"column:venue_id"`
	Venue        *Venue       `json:"venue,omitempty" gorm:"foreignKey:VenueID"`
	JobDevices   []JobDevice  `json:"job_devices,omitempty" gorm:"foreignKey:JobID"`
	JobPackages  []JobPackage `json:"job_packages,omitempty" gorm:"foreignKey:JobID;references:JobID"`
	DeviceCount  int          `json:"device_count" gorm:"-:all"`
}

func (Job) TableName() string {
	return "jobs"
}

func (j *Job) BeforeCreate(tx *gorm.DB) error {
	// Set a temporary job_code before insert to satisfy NOT NULL constraint
	// The actual code will be set in AfterCreate once we have the JobID
	if j.JobCode == "" {
		j.JobCode = "PENDING"
	}
	return nil
}

func (j *Job) AfterCreate(tx *gorm.DB) error {
	code := fmt.Sprintf("JOB%06d", j.JobID)
	j.JobCode = code
	return tx.Model(j).Update("job_code", code).Error
}

func (j *Job) BeforeUpdate(tx *gorm.DB) error {
	j.JobCode = fmt.Sprintf("JOB%06d", j.JobID)
	return nil
}

type Device struct {
	DeviceID            string      `json:"deviceID" gorm:"primaryKey;column:deviceid"`
	ProductID           *uint       `json:"productID" gorm:"column:productid"`
	Product             *Product    `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	SerialNumber        *string     `json:"serialnumber" gorm:"column:serialnumber"`
	PurchaseDate        *time.Time  `json:"purchaseDate" gorm:"column:purchasedate;type:date"`
	LastMaintenance     *time.Time  `json:"lastmaintenance" gorm:"column:lastmaintenance;type:date"`
	NextMaintenance     *time.Time  `json:"nextmaintenance" gorm:"column:nextmaintenance;type:date"`
	InsuranceNumber     *string     `json:"insurancenumber" gorm:"column:insurancenumber"`
	Status              string      `json:"status" gorm:"column:status;default:free"`
	InsuranceID         *uint       `json:"insuranceID" gorm:"column:insuranceid"`
	QRCode              *string     `json:"qrCode" gorm:"column:qr_code"`
	CurrentLocation     *string     `json:"currentLocation" gorm:"column:current_location"`
	GPSLatitude         *float64    `json:"gpsLatitude" gorm:"column:gps_latitude"`
	GPSLongitude        *float64    `json:"gpsLongitude" gorm:"column:gps_longitude"`
	ConditionRating     *float64    `json:"conditionRating" gorm:"column:condition_rating;default:5.0"`
	UsageHours          *float64    `json:"usageHours" gorm:"column:usage_hours;default:0.00"`
	TotalRevenue        *float64    `json:"totalRevenue" gorm:"column:total_revenue;default:0.00"`
	LastMaintenanceCost *float64    `json:"lastMaintenanceCost" gorm:"column:last_maintenance_cost"`
	Notes               *string     `json:"notes" gorm:"column:notes"`
	Barcode             *string     `json:"barcode" gorm:"column:barcode"`
	JobDevices          []JobDevice `json:"job_devices,omitempty" gorm:"-"`
}

func (Device) TableName() string {
	return "devices"
}

type Product struct {
	ProductID           uint     `json:"productID" gorm:"primaryKey;column:productid"`
	Name                string   `json:"name" gorm:"not null;column:name"`
	CategoryID          *uint    `json:"categoryID" gorm:"column:categoryid"`
	SubcategoryID       *string  `json:"subcategoryID" gorm:"column:subcategoryid"`
	SubbiercategoryID   *string  `json:"subbiercategoryID" gorm:"column:subbiercategoryid"`
	ManufacturerID      *uint    `json:"manufacturerID" gorm:"column:manufacturerid"`
	BrandID             *uint    `json:"brandID" gorm:"column:brandid"`
	Description         *string  `json:"description" gorm:"column:description"`
	MaintenanceInterval *uint    `json:"maintenanceInterval" gorm:"column:maintenanceinterval"`
	ItemCostPerDay      *float64 `json:"itemcostperday" gorm:"column:itemcostperday"`
	Weight              *float64 `json:"weight" gorm:"column:weight"`
	Height              *float64 `json:"height" gorm:"column:height"`
	Width               *float64 `json:"width" gorm:"column:width"`
	Depth               *float64 `json:"depth" gorm:"column:depth"`
	PowerConsumption    *float64 `json:"powerconsumption" gorm:"column:powerconsumption"`
	PosInCategory       *uint    `json:"pos_in_category" gorm:"column:pos_in_category"`

	// Accessory and Consumable fields
	IsAccessory    bool     `json:"is_accessory" gorm:"column:is_accessory;default:0;index"`
	IsConsumable   bool     `json:"is_consumable" gorm:"column:is_consumable;default:0;index"`
	CountTypeID    *uint    `json:"count_type_id" gorm:"column:count_type_id"`
	StockQuantity  *float64 `json:"stock_quantity" gorm:"column:stock_quantity;type:decimal(10,3)"`
	MinStockLevel  *float64 `json:"min_stock_level" gorm:"column:min_stock_level;type:decimal(10,3)"`
	GenericBarcode *string  `json:"generic_barcode" gorm:"column:generic_barcode;index"`
	PricePerUnit   *float64 `json:"price_per_unit" gorm:"column:price_per_unit;type:decimal(10,2)"`

	// Relations
	Category        *Category        `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:CategoryID"`
	Subcategory     *Subcategory     `json:"subcategory,omitempty" gorm:"foreignKey:SubcategoryID;references:SubcategoryID"`
	Subbiercategory *Subbiercategory `json:"subbiercategory,omitempty" gorm:"foreignKey:SubbiercategoryID;references:SubbiercategoryID"`
	Brand           *Brand           `json:"brand,omitempty" gorm:"foreignKey:BrandID"`
	Manufacturer    *Manufacturer    `json:"manufacturer,omitempty" gorm:"foreignKey:ManufacturerID"`
	CountType       *CountType       `json:"count_type,omitempty" gorm:"foreignKey:CountTypeID;references:CountTypeID"`
}

func (Product) TableName() string {
	return "products"
}

type Subcategory struct {
	SubcategoryID string   `json:"subcategoryID" gorm:"primaryKey;column:subcategoryid"`
	Name          string   `json:"name" gorm:"not null;column:name"`
	Abbreviation  string   `json:"abbreviation" gorm:"column:abbreviation"`
	CategoryID    uint     `json:"categoryID" gorm:"column:categoryid"`
	Category      Category `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:CategoryID"`
}

func (Subcategory) TableName() string {
	return "subcategories"
}

type Subbiercategory struct {
	SubbiercategoryID string      `json:"subbiercategoryID" gorm:"primaryKey;column:subbiercategoryid"`
	Name              string      `json:"name" gorm:"not null;column:name"`
	Abbreviation      string      `json:"abbreviation" gorm:"column:abbreviation"`
	SubcategoryID     string      `json:"subcategoryID" gorm:"column:subcategoryid"`
	Subcategory       Subcategory `json:"subcategory,omitempty" gorm:"foreignKey:SubcategoryID;references:SubcategoryID"`
}

func (Subbiercategory) TableName() string {
	return "subbiercategories"
}

type JobDevice struct {
	JobID         int        `json:"jobID" gorm:"primaryKey;column:jobid"`
	DeviceID      string     `json:"deviceID" gorm:"primaryKey;column:deviceid"`
	Job           Job        `json:"job,omitempty" gorm:"foreignKey:JobID"`
	Device        Device     `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
	CustomPrice   *float64   `json:"custom_price" gorm:"column:custom_price"`
	PackageID     *int       `json:"package_id" gorm:"column:package_id"`
	IsPackageItem bool       `json:"is_package_item" gorm:"column:is_package_item;default:false"`
	PackStatus    string     `json:"pack_status" gorm:"column:pack_status;default:pending"`
	PackTs        *time.Time `json:"pack_ts" gorm:"column:pack_ts"`
}

func (JobDevice) TableName() string {
	return "job_devices"
}

// JobWithDetails represents a job with aggregated information
type JobWithDetails struct {
	JobID         uint       `json:"jobID" gorm:"column:jobid"`
	JobCode       string     `json:"job_code" gorm:"column:job_code"`
	CustomerID    uint       `json:"customerID" gorm:"column:customerid"`
	StatusID      uint       `json:"statusID" gorm:"column:statusid"`
	JobCategoryID *uint      `json:"jobcategoryID" gorm:"column:jobcategoryid"`
	Description   *string    `json:"description" gorm:"column:description"`
	StartDate     *time.Time `json:"startDate" gorm:"column:startdate"`
	EndDate       *time.Time `json:"endDate" gorm:"column:enddate"`
	Revenue       float64    `json:"revenue" gorm:"column:revenue"`
	FinalRevenue  *float64   `json:"final_revenue" gorm:"column:final_revenue"`
	CustomerName  string     `json:"customer_name" gorm:"column:customer_name"`
	StatusName    string     `json:"status_name" gorm:"column:status_name"`
	CategoryName  *string    `json:"category_name" gorm:"column:category_name"`
	DeviceCount   int        `json:"device_count" gorm:"column:device_count"`
	TotalRevenue  float64    `json:"total_revenue" gorm:"column:total_revenue"`
}

// DeviceWithJobInfo represents a device with its current job assignment
type DeviceWithJobInfo struct {
	Device
	JobID      *uint   `json:"job_id"`
	JobTitle   *string `json:"job_title"`
	IsAssigned bool    `json:"is_assigned"`
}

// BulkScanRequest represents a request for bulk device scanning
type BulkScanRequest struct {
	JobID     uint     `json:"job_id" binding:"required"`
	DeviceIDs []string `json:"device_ids" binding:"required"`
	Price     float64  `json:"price"`
}

// ScanResult represents the result of a device scan operation
type ScanResult struct {
	DeviceID string  `json:"device_id"`
	Success  bool    `json:"success"`
	Message  string  `json:"message"`
	Device   *Device `json:"device,omitempty"`
}

// Additional models matching your database schema

type JobCategory struct {
	JobCategoryID uint    `json:"jobcategoryID" gorm:"primaryKey;column:jobcategoryid"`
	Name          string  `json:"name" gorm:"column:name"`
	Abbreviation  *string `json:"abbreviation" gorm:"column:abbreviation"`
}

func (JobCategory) TableName() string {
	return "jobcategory"
}

type Category struct {
	CategoryID   uint   `json:"categoryID" gorm:"primaryKey;column:categoryid"`
	Name         string `json:"name" gorm:"column:name"`
	Abbreviation string `json:"abbreviation" gorm:"column:abbreviation"`
}

func (Category) TableName() string {
	return "categories"
}

type Brand struct {
	BrandID        uint   `json:"brandID" gorm:"primaryKey;column:brandid"`
	Name           string `json:"name" gorm:"column:name"`
	ManufacturerID *uint  `json:"manufacturerID" gorm:"column:manufacturerid"`
}

func (Brand) TableName() string {
	return "brands"
}

type Manufacturer struct {
	ManufacturerID uint    `json:"manufacturerID" gorm:"primaryKey;column:manufacturerid"`
	Name           string  `json:"name" gorm:"column:name"`
	Website        *string `json:"website" gorm:"column:website"`
}

func (Manufacturer) TableName() string {
	return "manufacturer"
}

// FilterParams represents parameters for filtering jobs and devices
type FilterParams struct {
	StartDate    *time.Time `form:"start_date"`
	EndDate      *time.Time `form:"end_date"`
	CustomerID   *uint      `form:"customer_id"`
	StatusID     *uint      `form:"status_id"`
	MinRevenue   *float64   `form:"min_revenue"`
	MaxRevenue   *float64   `form:"max_revenue"`
	SearchTerm   string     `form:"search"`
	Category     string     `form:"category"`
	Available    *bool      `form:"available"`
	Limit        int        `form:"limit"`
	Offset       int        `form:"offset"`
	Connector1ID *uint      `form:"connector1"`
	Connector2ID *uint      `form:"connector2"`
	CableTypeID  *uint      `form:"cable_type"`
	MinLength    *float64   `form:"length_min"`
	MaxLength    *float64   `form:"length_max"`
	// Additional fields for optimized repository
	Page             int    `form:"page"`
	SortBy           string `form:"sort_by"`
	SortOrder        string `form:"sort_order"`
	Status           string `form:"status"`
	ProductID        *uint  `form:"product_id"`
	AssignmentStatus string `form:"assignment_status"`
	JobID            *uint  `form:"job_id"`
}

// DeviceAssignmentHistory represents the history of device assignments
type DeviceAssignmentHistory struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DeviceID     string         `json:"deviceID" gorm:"not null"`
	JobID        *uint          `json:"jobID" gorm:"index"`
	CustomerID   *uint          `json:"customerID" gorm:"index"`
	AssignedAt   time.Time      `json:"assignedAt" gorm:"not null"`
	UnassignedAt *time.Time     `json:"unassignedAt"`
	Duration     *time.Duration `json:"duration"`
	Notes        string         `json:"notes"`
	AssignedBy   string         `json:"assignedBy"`
	CreatedAt    time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (DeviceAssignmentHistory) TableName() string {
	return "device_assignment_history"
}

// User represents a user account for authentication
type User struct {
	UserID              uint       `json:"userID" gorm:"primaryKey;column:userid"`
	Username            string     `json:"username" gorm:"unique;not null;column:username"`
	Email               string     `json:"email" gorm:"unique;not null;column:email"`
	PasswordHash        string     `json:"-" gorm:"not null;column:password_hash"`
	FirstName           string     `json:"firstName" gorm:"column:first_name"`
	LastName            string     `json:"lastName" gorm:"column:last_name"`
	IsAdmin             bool       `json:"isAdmin" gorm:"default:false;column:is_admin"`
	IsActive            bool       `json:"isActive" gorm:"default:true;column:is_active"`
	CreatedAt           time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt           time.Time  `json:"updatedAt" gorm:"column:updated_at"`
	LastLogin           *time.Time `json:"lastLogin" gorm:"column:last_login"`
	ForcePasswordChange bool       `json:"forcePasswordChange" gorm:"default:false;column:force_password_change"`
}

func (User) TableName() string {
	return "users"
}

func (u User) GetDisplayName() string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.LastName != "" {
		return u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	return u.Username
}

// Session represents a user session
type Session struct {
	SessionID string    `json:"sessionID" gorm:"primaryKey;column:session_id"`
	UserID    uint      `json:"userID" gorm:"not null;column:user_id"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"not null;column:expires_at"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (Session) TableName() string {
	return "sessions"
}

// UserPreferences represents global user profile settings
type UserPreferences struct {
	PreferenceID uint `json:"preferenceID" gorm:"primaryKey;column:preference_id"`
	UserID       uint `json:"userID" gorm:"not null;unique;column:user_id"`
	User         User `json:"user,omitempty" gorm:"-"`

	// Display Preferences
	Language   string `json:"language" gorm:"not null;default:'de';column:language"`
	Theme      string `json:"theme" gorm:"not null;default:'dark';column:theme"`
	TimeZone   string `json:"timeZone" gorm:"not null;default:'Europe/Berlin';column:time_zone"`
	DateFormat string `json:"dateFormat" gorm:"not null;default:'DD.MM.YYYY';column:date_format"`
	TimeFormat string `json:"timeFormat" gorm:"not null;default:'24h';column:time_format"`

	// Notification Preferences
	EmailNotifications       bool `json:"emailNotifications" gorm:"not null;default:true;column:email_notifications"`
	SystemNotifications      bool `json:"systemNotifications" gorm:"not null;default:true;column:system_notifications"`
	JobStatusNotifications   bool `json:"jobStatusNotifications" gorm:"not null;default:true;column:job_status_notifications"`
	DeviceAlertNotifications bool `json:"deviceAlertNotifications" gorm:"not null;default:true;column:device_alert_notifications"`

	// Interface Preferences
	ItemsPerPage        int    `json:"itemsPerPage" gorm:"not null;default:25;column:items_per_page"`
	DefaultView         string `json:"defaultView" gorm:"not null;default:'list';column:default_view"`
	ShowAdvancedOptions bool   `json:"showAdvancedOptions" gorm:"not null;default:false;column:show_advanced_options"`
	AutoSaveEnabled     bool   `json:"autoSaveEnabled" gorm:"not null;default:true;column:auto_save_enabled"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (UserPreferences) TableName() string {
	return "user_preferences"
}

type Case struct {
	CaseID      uint         `json:"caseID" gorm:"primaryKey;column:caseid"`
	Name        string       `json:"name" gorm:"not null;column:name"`
	Description *string      `json:"description" gorm:"column:description"`
	Weight      *float64     `json:"weight" gorm:"column:weight"`
	Width       *float64     `json:"width" gorm:"column:width"`
	Height      *float64     `json:"height" gorm:"column:height"`
	Depth       *float64     `json:"depth" gorm:"column:depth"`
	Status      string       `json:"status" gorm:"not null;column:status;default:free"`
	Devices     []DeviceCase `json:"devices,omitempty" gorm:"foreignKey:CaseID"`
	DeviceCount int          `json:"device_count" gorm:"-:all"`
}

func (Case) TableName() string {
	return "cases"
}

type DeviceCase struct {
	CaseID   uint   `json:"caseID" gorm:"primaryKey;column:caseid"`
	DeviceID string `json:"deviceID" gorm:"primaryKey;column:deviceid"`
	Case     Case   `json:"case,omitempty" gorm:"foreignKey:CaseID"`
	Device   Device `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

func (DeviceCase) TableName() string {
	return "devicescases"
}

// Cable management models
type Cable struct {
	CableID    int      `json:"cableID" gorm:"primaryKey;column:cableID"`
	Connector1 int      `json:"connector1" gorm:"not null;column:connector1"`
	Connector2 int      `json:"connector2" gorm:"not null;column:connector2"`
	Type       int      `json:"typ" gorm:"not null;column:typ"`
	Length     float64  `json:"length" gorm:"not null;column:length"`
	MM2        *float64 `json:"mm2" gorm:"column:mm2"`
	Name       *string  `json:"name" gorm:"column:name"`

	// Relationships
	Connector1Info *CableConnector `json:"connector1_info,omitempty" gorm:"foreignKey:Connector1;references:CableConnectorsID"`
	Connector2Info *CableConnector `json:"connector2_info,omitempty" gorm:"foreignKey:Connector2;references:CableConnectorsID"`
	TypeInfo       *CableType      `json:"type_info,omitempty" gorm:"foreignKey:Type;references:CableTypesID"`
}

func (Cable) TableName() string {
	return "cables"
}

// GetMM2Display returns the formatted MM2 value or "-" if nil
func (c Cable) GetMM2Display() string {
	if c.MM2 == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f mm²", *c.MM2)
}

// GetMM2Value returns the MM2 value or empty string if nil
func (c Cable) GetMM2Value() string {
	if c.MM2 == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *c.MM2)
}

type CableConnector struct {
	CableConnectorsID int     `json:"cable_connectorsID" gorm:"primaryKey;column:cable_connectorsID"`
	Name              string  `json:"name" gorm:"not null;column:name"`
	Abbreviation      *string `json:"abbreviation" gorm:"column:abbreviation"`
	Gender            *string `json:"gender" gorm:"column:gender"`
}

func (CableConnector) TableName() string {
	return "cable_connectors"
}

type CableType struct {
	CableTypesID int    `json:"cable_typesID" gorm:"primaryKey;column:cable_typesID"`
	Name         string `json:"name" gorm:"not null;column:name"`
}

func (CableType) TableName() string {
	return "cable_types"
}

// CableGroup represents grouped cables with same specifications
type CableGroup struct {
	Type       int      `json:"typ"`
	Connector1 int      `json:"connector1"`
	Connector2 int      `json:"connector2"`
	Length     float64  `json:"length"`
	MM2        *float64 `json:"mm2"`
	Name       *string  `json:"name"`
	Count      int      `json:"count"`

	// Relationships
	Connector1Info *CableConnector `json:"connector1_info,omitempty"`
	Connector2Info *CableConnector `json:"connector2_info,omitempty"`
	TypeInfo       *CableType      `json:"type_info,omitempty"`

	// Sample cable IDs from this group
	CableIDs []int `json:"cable_ids,omitempty"`
}

// GetMM2Display returns the formatted MM2 value or "-" if nil
func (cg CableGroup) GetMM2Display() string {
	if cg.MM2 == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f mm²", *cg.MM2)
}

// ================================================================
// AUTHENTICATION MODELS - Passkeys and 2FA
// ================================================================

// UserPasskey represents a WebAuthn passkey for a user
type UserPasskey struct {
	PasskeyID    uint       `json:"passkeyID" gorm:"primaryKey;column:passkey_id"`
	UserID       uint       `json:"userID" gorm:"not null;column:user_id"`
	User         User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Name         string     `json:"name" gorm:"not null;column:name"`
	CredentialID string     `json:"credentialID" gorm:"not null;unique;column:credential_id"`
	PublicKey    []byte     `json:"publicKey" gorm:"column:public_key"`
	SignCount    uint32     `json:"signCount" gorm:"default:0;column:sign_count"`
	AAGUID       []byte     `json:"aaguid" gorm:"column:aaguid"`
	IsActive     bool       `json:"isActive" gorm:"default:true;column:is_active"`
	LastUsed     *time.Time `json:"lastUsed" gorm:"column:last_used"`
	CreatedAt    time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" gorm:"column:updated_at"`
}

func (UserPasskey) TableName() string {
	return "user_passkeys"
}

// User2FA represents TOTP 2FA settings for a user
type User2FA struct {
	TwoFAID     uint       `json:"twoFAID" gorm:"primaryKey;column:two_fa_id"`
	UserID      uint       `json:"userID" gorm:"not null;unique;column:user_id"`
	User        User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Secret      string     `json:"secret" gorm:"not null;column:secret"`
	QRCodeURL   string     `json:"qrCodeURL" gorm:"column:qr_code_url"`
	IsEnabled   bool       `json:"isEnabled" gorm:"default:false;column:is_enabled"`
	IsVerified  bool       `json:"isVerified" gorm:"default:false;column:is_verified"`
	BackupCodes []string   `json:"backupCodes" gorm:"type:json;column:backup_codes"`
	LastUsed    *time.Time `json:"lastUsed" gorm:"column:last_used"`
	CreatedAt   time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" gorm:"column:updated_at"`
}

func (User2FA) TableName() string {
	return "user_2fa"
}

// AuthenticationAttempt logs authentication attempts for security monitoring
type AuthenticationAttempt struct {
	AttemptID     uint      `json:"attemptID" gorm:"primaryKey;column:attempt_id"`
	UserID        *uint     `json:"userID" gorm:"column:user_id"`
	User          *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Method        string    `json:"method" gorm:"not null;column:method"` // password, passkey, 2fa, backup_code
	IPAddress     string    `json:"ipAddress" gorm:"not null;column:ip_address"`
	UserAgent     string    `json:"userAgent" gorm:"column:user_agent"`
	Success       bool      `json:"success" gorm:"not null;column:success"`
	FailureReason *string   `json:"failureReason" gorm:"column:failure_reason"`
	PasskeyID     *uint     `json:"passkeyID" gorm:"column:passkey_id"`
	AttemptedAt   time.Time `json:"attemptedAt" gorm:"column:attempted_at"`
}

func (AuthenticationAttempt) TableName() string {
	return "authentication_attempts"
}

// WebAuthnSession represents temporary WebAuthn session data
type WebAuthnSession struct {
	SessionID   string    `json:"sessionID" gorm:"primaryKey;column:session_id"`
	UserID      uint      `json:"userID" gorm:"not null;column:user_id"`
	Challenge   string    `json:"challenge" gorm:"not null;column:challenge"`
	SessionType string    `json:"sessionType" gorm:"not null;column:session_type"` // registration, authentication
	SessionData string    `json:"sessionData" gorm:"type:text;column:session_data"`
	ExpiresAt   time.Time `json:"expiresAt" gorm:"not null;column:expires_at"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (WebAuthnSession) TableName() string {
	return "webauthn_sessions"
}

// JobDeviceEvent represents events in the pack workflow
type JobDeviceEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey;column:id"`
	JobID     uint      `json:"jobID" gorm:"column:jobid"`
	DeviceID  string    `json:"deviceID" gorm:"column:deviceid"`
	EventType string    `json:"event_type" gorm:"column:event_type"`
	Actor     *string   `json:"actor" gorm:"column:actor"`
	Timestamp time.Time `json:"timestamp" gorm:"column:timestamp;default:CURRENT_TIMESTAMP"`
	Metadata  *string   `json:"metadata" gorm:"column:metadata;type:json"`
}

func (JobDeviceEvent) TableName() string {
	return "job_device_events"
}

// ProductImage represents product images
type ProductImage struct {
	ImageID      uint      `json:"imageID" gorm:"primaryKey;column:imageID"`
	ProductID    uint      `json:"productID" gorm:"column:productid"`
	Filename     string    `json:"filename" gorm:"column:filename"`
	OriginalName *string   `json:"original_name" gorm:"column:original_name"`
	FilePath     string    `json:"file_path" gorm:"column:file_path"`
	FileSize     *uint64   `json:"file_size" gorm:"column:file_size"`
	MimeType     *string   `json:"mime_type" gorm:"column:mime_type"`
	IsPrimary    bool      `json:"is_primary" gorm:"column:is_primary;default:false"`
	AltText      *string   `json:"alt_text" gorm:"column:alt_text"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

// ScanBoardDevice represents a device in the scan board view
type ScanBoardDevice struct {
	DeviceID       string  `json:"deviceID"`
	ProductName    string  `json:"productName"`
	PackStatus     string  `json:"packStatus"`
	BarcodePayload string  `json:"barcodePayload"`
	ImageURL       *string `json:"imageUrl,omitempty"`
}

// ScanRequest represents a scan request for the pack workflow
type ScanRequest struct {
	DeviceID       *string `json:"deviceID,omitempty"`
	BarcodePayload *string `json:"barcodePayload,omitempty"`
}

// FinishPackRequest represents a request to finish packing a job
type FinishPackRequest struct {
	Force bool `json:"force"` // Force finish even if items are missing
}

// FinishPackResponse represents the response from finish pack endpoint
type FinishPackResponse struct {
	Success      bool     `json:"success"`
	MissingItems []string `json:"missingItems,omitempty"`
	Message      string   `json:"message"`
}
