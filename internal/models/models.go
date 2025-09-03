package models

import (
	"fmt"
	"time"
)

type Customer struct {
	CustomerID   uint      `json:"customerID" gorm:"primaryKey;column:customerID"`
	CompanyName  *string   `json:"companyname" gorm:"column:companyname"`
	LastName     *string   `json:"lastname" gorm:"column:lastname"`
	FirstName    *string   `json:"firstname" gorm:"column:firstname"`
	Street       *string   `json:"street" gorm:"column:street"`
	HouseNumber  *string   `json:"housenumber" gorm:"column:housenumber"`
	ZIP          *string   `json:"ZIP" gorm:"column:ZIP"`
	City         *string   `json:"city" gorm:"column:city"`
	FederalState *string   `json:"federalstate" gorm:"column:federalstate"`
	Country      *string   `json:"country" gorm:"column:country"`
	PhoneNumber  *string   `json:"phonenumber" gorm:"column:phonenumber"`
	Email        *string   `json:"email" gorm:"column:email"`
	CustomerType *string   `json:"customertype" gorm:"column:customertype"`
	Notes        *string   `json:"notes" gorm:"column:notes"`
	Jobs         []Job     `json:"jobs,omitempty" gorm:"-"`
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
	StatusID uint   `json:"statusID" gorm:"primaryKey;column:statusID"`
	Status   string `json:"status" gorm:"not null;column:status"`
	Jobs     []Job  `json:"jobs,omitempty" gorm:"-"`
}

func (Status) TableName() string {
	return "status"
}

type Job struct {
	JobID           uint        `json:"jobID" gorm:"primaryKey;column:jobID"`
	CustomerID      uint        `json:"customerID" gorm:"not null;column:customerID"`
	Customer        Customer    `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	StatusID        uint        `json:"statusID" gorm:"not null;column:statusID"`
	Status          Status      `json:"status,omitempty" gorm:"foreignKey:StatusID"`
	JobCategoryID   *uint       `json:"jobcategoryID" gorm:"column:jobcategoryID"`
	Description     *string     `json:"description" gorm:"column:description"`
	Discount        float64     `json:"discount" gorm:"column:discount;default:0"`
	DiscountType    string      `json:"discount_type" gorm:"column:discount_type;default:amount"`
	Revenue         float64     `json:"revenue" gorm:"column:revenue;default:0"`
	FinalRevenue    *float64    `json:"final_revenue" gorm:"column:final_revenue"`
	StartDate       *time.Time  `json:"startDate" gorm:"column:startDate;type:date"`
	EndDate         *time.Time  `json:"endDate" gorm:"column:endDate;type:date"`
	JobDevices      []JobDevice `json:"job_devices,omitempty" gorm:"foreignKey:JobID"`
	DeviceCount     int         `json:"device_count" gorm:"-:all"`
}

func (Job) TableName() string {
	return "jobs"
}

type Device struct {
	DeviceID             string      `json:"deviceID" gorm:"primaryKey;column:deviceID"`
	ProductID            *uint       `json:"productID" gorm:"column:productID"`
	Product              *Product    `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ProductID"`
	SerialNumber         *string     `json:"serialnumber" gorm:"column:serialnumber"`
	PurchaseDate         *time.Time  `json:"purchaseDate" gorm:"column:purchaseDate;type:date"`
	LastMaintenance      *time.Time  `json:"lastmaintenance" gorm:"column:lastmaintenance;type:date"`
	NextMaintenance      *time.Time  `json:"nextmaintenance" gorm:"column:nextmaintenance;type:date"`
	InsuranceNumber      *string     `json:"insurancenumber" gorm:"column:insurancenumber"`
	Status               string      `json:"status" gorm:"column:status;default:free"`
	InsuranceID          *uint       `json:"insuranceID" gorm:"column:insuranceID"`
	QRCode               *string     `json:"qrCode" gorm:"column:qr_code"`
	CurrentLocation      *string     `json:"currentLocation" gorm:"column:current_location"`
	GPSLatitude          *float64    `json:"gpsLatitude" gorm:"column:gps_latitude"`
	GPSLongitude         *float64    `json:"gpsLongitude" gorm:"column:gps_longitude"`
	ConditionRating      *float64    `json:"conditionRating" gorm:"column:condition_rating;default:5.0"`
	UsageHours           *float64    `json:"usageHours" gorm:"column:usage_hours;default:0.00"`
	TotalRevenue         *float64    `json:"totalRevenue" gorm:"column:total_revenue;default:0.00"`
	LastMaintenanceCost  *float64    `json:"lastMaintenanceCost" gorm:"column:last_maintenance_cost"`
	Notes                *string     `json:"notes" gorm:"column:notes"`
	Barcode              *string     `json:"barcode" gorm:"column:barcode"`
	JobDevices           []JobDevice `json:"job_devices,omitempty" gorm:"-"`
}

func (Device) TableName() string {
	return "devices"
}

type Product struct {
	ProductID             uint     `json:"productID" gorm:"primaryKey;column:productID"`
	Name                  string   `json:"name" gorm:"not null;column:name"`
	CategoryID            *uint    `json:"categoryID" gorm:"column:categoryID"`
	SubcategoryID         *string  `json:"subcategoryID" gorm:"column:subcategoryID"`
	SubbiercategoryID     *string  `json:"subbiercategoryID" gorm:"column:subbiercategoryID"`
	ManufacturerID        *uint    `json:"manufacturerID" gorm:"column:manufacturerID"`
	BrandID               *uint    `json:"brandID" gorm:"column:brandID"`
	Description           *string  `json:"description" gorm:"column:description"`
	MaintenanceInterval   *uint    `json:"maintenanceInterval" gorm:"column:maintenanceInterval"`
	ItemCostPerDay        *float64 `json:"itemcostperday" gorm:"column:itemcostperday"`
	Weight                *float64 `json:"weight" gorm:"column:weight"`
	Height                *float64 `json:"height" gorm:"column:height"`
	Width                 *float64 `json:"width" gorm:"column:width"`
	Depth                 *float64 `json:"depth" gorm:"column:depth"`
	PowerConsumption      *float64     `json:"powerconsumption" gorm:"column:powerconsumption"`
	PosInCategory         *uint        `json:"pos_in_category" gorm:"column:pos_in_category"`
	Category              *Category       `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:CategoryID"`
	Subcategory           *Subcategory    `json:"subcategory,omitempty" gorm:"foreignKey:SubcategoryID;references:SubcategoryID"`
	Subbiercategory       *Subbiercategory `json:"subbiercategory,omitempty" gorm:"foreignKey:SubbiercategoryID;references:SubbiercategoryID"`
	Brand                 *Brand          `json:"brand,omitempty" gorm:"foreignKey:BrandID"`
	Manufacturer          *Manufacturer   `json:"manufacturer,omitempty" gorm:"foreignKey:ManufacturerID"`
}

func (Product) TableName() string {
	return "products"
}


type Subcategory struct {
	SubcategoryID string  `json:"subcategoryID" gorm:"primaryKey;column:subcategoryID"`
	Name          string  `json:"name" gorm:"not null;column:name"`
	Abbreviation  string  `json:"abbreviation" gorm:"column:abbreviation"`
	CategoryID    uint    `json:"categoryID" gorm:"column:categoryID"`
	Category      Category `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:CategoryID"`
}

func (Subcategory) TableName() string {
	return "subcategories"
}

type Subbiercategory struct {
	SubbiercategoryID string `json:"subbiercategoryID" gorm:"primaryKey;column:subbiercategoryID"`
	Name              string `json:"name" gorm:"not null;column:name"`
	Abbreviation      string `json:"abbreviation" gorm:"column:abbreviation"`
	SubcategoryID     string `json:"subcategoryID" gorm:"column:subcategoryID"`
	Subcategory       Subcategory `json:"subcategory,omitempty" gorm:"foreignKey:SubcategoryID;references:SubcategoryID"`
}

func (Subbiercategory) TableName() string {
	return "subbiercategories"
}

type JobDevice struct {
	JobID       uint     `json:"jobID" gorm:"primaryKey;column:jobID"`
	DeviceID    string   `json:"deviceID" gorm:"primaryKey;column:deviceID"`
	Job         Job      `json:"job,omitempty" gorm:"foreignKey:JobID"`
	Device      Device   `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
	CustomPrice *float64 `json:"custom_price" gorm:"column:custom_price"`
}

func (JobDevice) TableName() string {
	return "jobdevices"
}

// JobWithDetails represents a job with aggregated information
type JobWithDetails struct {
	JobID        uint       `json:"jobID" gorm:"column:jobID"`
	CustomerID   uint       `json:"customerID" gorm:"column:customerID"`
	StatusID     uint       `json:"statusID" gorm:"column:statusID"`
	Description  *string    `json:"description" gorm:"column:description"`
	StartDate    *time.Time `json:"startDate" gorm:"column:startDate"`
	EndDate      *time.Time `json:"endDate" gorm:"column:endDate"`
	Revenue      float64    `json:"revenue" gorm:"column:revenue"`
	FinalRevenue *float64   `json:"final_revenue" gorm:"column:final_revenue"`
	CustomerName string     `json:"customer_name" gorm:"column:customer_name"`
	StatusName   string     `json:"status_name" gorm:"column:status_name"`
	DeviceCount  int        `json:"device_count" gorm:"column:device_count"`
	TotalRevenue float64    `json:"total_revenue" gorm:"column:total_revenue"`
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
	JobCategoryID uint    `json:"jobcategoryID" gorm:"primaryKey;column:jobcategoryID"`
	Name          string  `json:"name" gorm:"column:name"`
	Abbreviation  *string `json:"abbreviation" gorm:"column:abbreviation"`
}

func (JobCategory) TableName() string {
	return "jobCategory"
}

type Category struct {
	CategoryID   uint    `json:"categoryID" gorm:"primaryKey;column:categoryID"`
	Name         string  `json:"name" gorm:"column:name"`
	Abbreviation string  `json:"abbreviation" gorm:"column:abbreviation"`
}

func (Category) TableName() string {
	return "categories"
}

type Brand struct {
	BrandID        uint    `json:"brandID" gorm:"primaryKey;column:brandID"`
	Name           string  `json:"name" gorm:"column:name"`
	ManufacturerID *uint   `json:"manufacturerID" gorm:"column:manufacturerID"`
}

func (Brand) TableName() string {
	return "brands"
}

type Manufacturer struct {
	ManufacturerID uint    `json:"manufacturerID" gorm:"primaryKey;column:manufacturerID"`
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
	// Additional fields for optimized repository
	Page               int    `form:"page"`
	SortBy             string `form:"sort_by"`
	SortOrder          string `form:"sort_order"`
	Status             string `form:"status"`
	ProductID          *uint  `form:"product_id"`
	AssignmentStatus   string `form:"assignment_status"`
	JobID              *uint  `form:"job_id"`
}

// DeviceAssignmentHistory represents the history of device assignments
type DeviceAssignmentHistory struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	DeviceID     string    `json:"deviceID" gorm:"not null"`
	JobID        *uint     `json:"jobID" gorm:"index"`
	CustomerID   *uint     `json:"customerID" gorm:"index"`
	AssignedAt   time.Time `json:"assignedAt" gorm:"not null"`
	UnassignedAt *time.Time `json:"unassignedAt"`
	Duration     *time.Duration `json:"duration"`
	Notes        string    `json:"notes"`
	AssignedBy   string    `json:"assignedBy"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (DeviceAssignmentHistory) TableName() string {
	return "device_assignment_history"
}

// User represents a user account for authentication
type User struct {
	UserID       uint      `json:"userID" gorm:"primaryKey;column:userID"`
	Username     string    `json:"username" gorm:"unique;not null;column:username"`
	Email        string    `json:"email" gorm:"unique;not null;column:email"`
	PasswordHash string    `json:"-" gorm:"not null;column:password_hash"`
	FirstName    string    `json:"firstName" gorm:"column:first_name"`
	LastName     string    `json:"lastName" gorm:"column:last_name"`
	IsActive     bool      `json:"isActive" gorm:"default:true;column:is_active"`
	CreatedAt    time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"column:updated_at"`
	LastLogin    *time.Time `json:"lastLogin" gorm:"column:last_login"`
}

func (User) TableName() string {
	return "users"
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
	PreferenceID uint      `json:"preferenceID" gorm:"primaryKey;column:preference_id"`
	UserID       uint      `json:"userID" gorm:"not null;unique;column:user_id"`
	User         User      `json:"user,omitempty" gorm:"-"`
	
	// Display Preferences
	Language     string    `json:"language" gorm:"not null;default:'de';column:language"`
	Theme        string    `json:"theme" gorm:"not null;default:'dark';column:theme"`
	TimeZone     string    `json:"timeZone" gorm:"not null;default:'Europe/Berlin';column:time_zone"`
	DateFormat   string    `json:"dateFormat" gorm:"not null;default:'DD.MM.YYYY';column:date_format"`
	TimeFormat   string    `json:"timeFormat" gorm:"not null;default:'24h';column:time_format"`
	
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
	CaseID      uint            `json:"caseID" gorm:"primaryKey;column:caseID"`
	Name        string          `json:"name" gorm:"not null;column:name"`
	Description *string         `json:"description" gorm:"column:description"`
	Weight      *float64        `json:"weight" gorm:"column:weight"`
	Width       *float64        `json:"width" gorm:"column:width"`
	Height      *float64        `json:"height" gorm:"column:height"`
	Depth       *float64        `json:"depth" gorm:"column:depth"`
	Status      string          `json:"status" gorm:"not null;column:status;default:free"`
	Devices     []DeviceCase    `json:"devices,omitempty" gorm:"foreignKey:CaseID"`
	DeviceCount int             `json:"device_count" gorm:"-:all"`
}

func (Case) TableName() string {
	return "cases"
}

type DeviceCase struct {
	CaseID   uint   `json:"caseID" gorm:"primaryKey;column:caseID"`
	DeviceID string `json:"deviceID" gorm:"primaryKey;column:deviceID"`
	Case     Case   `json:"case,omitempty" gorm:"foreignKey:CaseID"`
	Device   Device `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

func (DeviceCase) TableName() string {
	return "devicescases"
}

// Cable management models
type Cable struct {
	CableID    int             `json:"cableID" gorm:"primaryKey;column:cableID"`
	Connector1 int             `json:"connector1" gorm:"not null;column:connector1"`
	Connector2 int             `json:"connector2" gorm:"not null;column:connector2"`
	Type       int             `json:"typ" gorm:"not null;column:typ"`
	Length     float64         `json:"length" gorm:"not null;column:length"`
	MM2        *float64        `json:"mm2" gorm:"column:mm2"`
	Name       *string         `json:"name" gorm:"column:name"`
	
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
	Type       int             `json:"typ"`
	Connector1 int             `json:"connector1"`
	Connector2 int             `json:"connector2"`
	Length     float64         `json:"length"`
	MM2        *float64        `json:"mm2"`
	Name       *string         `json:"name"`
	Count      int             `json:"count"`
	
	// Relationships
	Connector1Info *CableConnector `json:"connector1_info,omitempty"`
	Connector2Info *CableConnector `json:"connector2_info,omitempty"`
	TypeInfo       *CableType      `json:"type_info,omitempty"`
	
	// Sample cable IDs from this group
	CableIDs       []int           `json:"cable_ids,omitempty"`
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
	PasskeyID    uint      `json:"passkeyID" gorm:"primaryKey;column:passkey_id"`
	UserID       uint      `json:"userID" gorm:"not null;column:user_id"`
	User         User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Name         string    `json:"name" gorm:"not null;column:name"`
	CredentialID string    `json:"credentialID" gorm:"not null;unique;column:credential_id"`
	PublicKey    []byte    `json:"publicKey" gorm:"column:public_key"`
	SignCount    uint32    `json:"signCount" gorm:"default:0;column:sign_count"`
	AAGUID       []byte    `json:"aaguid" gorm:"column:aaguid"`
	IsActive     bool      `json:"isActive" gorm:"default:true;column:is_active"`
	LastUsed     *time.Time `json:"lastUsed" gorm:"column:last_used"`
	CreatedAt    time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (UserPasskey) TableName() string {
	return "user_passkeys"
}

// User2FA represents TOTP 2FA settings for a user
type User2FA struct {
	TwoFAID     uint      `json:"twoFAID" gorm:"primaryKey;column:two_fa_id"`
	UserID      uint      `json:"userID" gorm:"not null;unique;column:user_id"`
	User        User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Secret      string    `json:"secret" gorm:"not null;column:secret"`
	QRCodeURL   string    `json:"qrCodeURL" gorm:"column:qr_code_url"`
	IsEnabled   bool      `json:"isEnabled" gorm:"default:false;column:is_enabled"`
	IsVerified  bool      `json:"isVerified" gorm:"default:false;column:is_verified"`
	BackupCodes []string  `json:"backupCodes" gorm:"type:json;column:backup_codes"`
	LastUsed    *time.Time `json:"lastUsed" gorm:"column:last_used"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (User2FA) TableName() string {
	return "user_2fa"
}

// AuthenticationAttempt logs authentication attempts for security monitoring
type AuthenticationAttempt struct {
	AttemptID    uint      `json:"attemptID" gorm:"primaryKey;column:attempt_id"`
	UserID       *uint     `json:"userID" gorm:"column:user_id"`
	User         *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Method       string    `json:"method" gorm:"not null;column:method"` // password, passkey, 2fa, backup_code
	IPAddress    string    `json:"ipAddress" gorm:"not null;column:ip_address"`
	UserAgent    string    `json:"userAgent" gorm:"column:user_agent"`
	Success      bool      `json:"success" gorm:"not null;column:success"`
	FailureReason *string  `json:"failureReason" gorm:"column:failure_reason"`
	PasskeyID    *uint     `json:"passkeyID" gorm:"column:passkey_id"`
	AttemptedAt  time.Time `json:"attemptedAt" gorm:"column:attempted_at"`
}

func (AuthenticationAttempt) TableName() string {
	return "authentication_attempts"
}

// WebAuthnSession represents temporary WebAuthn session data
type WebAuthnSession struct {
	SessionID     string    `json:"sessionID" gorm:"primaryKey;column:session_id"`
	UserID        uint      `json:"userID" gorm:"not null;column:user_id"`
	Challenge     string    `json:"challenge" gorm:"not null;column:challenge"`
	SessionType   string    `json:"sessionType" gorm:"not null;column:session_type"` // registration, authentication
	SessionData   string    `json:"sessionData" gorm:"type:text;column:session_data"`
	ExpiresAt     time.Time `json:"expiresAt" gorm:"not null;column:expires_at"`
	CreatedAt     time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (WebAuthnSession) TableName() string {
	return "webauthn_sessions"
}
