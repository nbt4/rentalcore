package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BrandingConfig mirrors the branding_config DB row.
type BrandingConfig struct {
	CompanyName     string `json:"companyName"`
	BrandName       string `json:"brandName"`
	LogoSidebar     string `json:"sidebarLogo"`
	LogoLogin       string `json:"loginLogo"`
	FaviconPath     string `json:"faviconPath"`
	LogoSizeSidebar int16  `json:"logoSizeSidebar"`
	LogoSizeLogin   int16  `json:"logoSizeLogin"`
}

type brandingRecord struct {
	ID                   uint    `gorm:"primaryKey;column:id"`
	CompanyName          string  `gorm:"column:company_name"`
	BrandName            string  `gorm:"column:brand_name"`
	LogoCoresSidebar     *string `gorm:"column:logo_cores_sidebar"`
	LogoCoresLogin       *string `gorm:"column:logo_cores_login"`
	LogoRentalSidebar    *string `gorm:"column:logo_rental_sidebar"`
	LogoRentalLogin      *string `gorm:"column:logo_rental_login"`
	LogoWarehouseSidebar *string `gorm:"column:logo_warehouse_sidebar"`
	LogoWarehouseLogin   *string `gorm:"column:logo_warehouse_login"`
	LogoPlannerSidebar   *string `gorm:"column:logo_planner_sidebar"`
	LogoPlannerLogin     *string `gorm:"column:logo_planner_login"`
	FaviconPath          *string `gorm:"column:favicon_path"`
	FaviconCores          *string `gorm:"column:favicon_cores"`
	FaviconRental         *string `gorm:"column:favicon_rental"`
	FaviconWarehouse      *string `gorm:"column:favicon_warehouse"`
	FaviconPlanner        *string `gorm:"column:favicon_planner"`
	LogoSizeSidebar      int16     `gorm:"column:logo_size_sidebar"`
	LogoSizeLogin        int16     `gorm:"column:logo_size_login"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (brandingRecord) TableName() string { return "branding_config" }

// cacheBuster appends a version query param to force browser re-fetch on change.
func cacheBuster(path string, updatedAt time.Time) string {
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s?v=%d", path, updatedAt.Unix())
}

// BrandingService reads branding_config directly from the shared PostgreSQL DB.
// No caching — every call hits the DB for instant updates.
type BrandingService struct {
	db      *gorm.DB
	service string
}

func NewBrandingService(db *gorm.DB, service string) *BrandingService {
	return &BrandingService{db: db, service: service}
}

func (s *BrandingService) GetConfig() BrandingConfig {
	var rec brandingRecord
	if err := s.db.First(&rec, 1).Error; err != nil {
		return BrandingConfig{
			CompanyName:     s.defaultName(),
			LogoSizeSidebar: 100,
			LogoSizeLogin:   100,
		}
	}

	cfg := BrandingConfig{
		CompanyName:     s.coalesceName(rec.CompanyName),
		BrandName:       rec.BrandName,
		FaviconPath:     cacheBuster(s.faviconFor(rec), rec.UpdatedAt),
		LogoSizeSidebar: s.coalesceSize(rec.LogoSizeSidebar),
		LogoSizeLogin:   s.coalesceSize(rec.LogoSizeLogin),
	}

	switch s.service {
	case "cores":
		cfg.LogoSidebar = cacheBuster(s.deref(rec.LogoCoresSidebar), rec.UpdatedAt)
		cfg.LogoLogin = cacheBuster(s.deref(rec.LogoCoresLogin), rec.UpdatedAt)
	case "rental":
		cfg.LogoSidebar = cacheBuster(s.deref(rec.LogoRentalSidebar), rec.UpdatedAt)
		cfg.LogoLogin = cacheBuster(s.deref(rec.LogoRentalLogin), rec.UpdatedAt)
	case "warehouse":
		cfg.LogoSidebar = cacheBuster(s.deref(rec.LogoWarehouseSidebar), rec.UpdatedAt)
		cfg.LogoLogin = cacheBuster(s.deref(rec.LogoWarehouseLogin), rec.UpdatedAt)
	case "planner":
		cfg.LogoSidebar = cacheBuster(s.deref(rec.LogoPlannerSidebar), rec.UpdatedAt)
		cfg.LogoLogin = cacheBuster(s.deref(rec.LogoPlannerLogin), rec.UpdatedAt)
	}

	return cfg
}

func (s *BrandingService) faviconFor(rec brandingRecord) string {
	switch s.service {
	case "cores":
		return s.deref(rec.FaviconCores)
	case "rental":
		return s.deref(rec.FaviconRental)
	case "warehouse":
		return s.deref(rec.FaviconWarehouse)
	case "planner":
		return s.deref(rec.FaviconPlanner)
	}
	// Fallback: legacy global favicon
	return s.deref(rec.FaviconPath)
}

func (s *BrandingService) defaultName() string {
	switch s.service {
	case "rental":
		return "RentalCore"
	case "warehouse":
		return "WarehouseCore"
	case "planner":
		return "PlannerCore"
	default:
		return ""
	}
}

func (s *BrandingService) coalesceName(dbName string) string {
	if dbName != "" {
		return dbName
	}
	return s.defaultName()
}

func (s *BrandingService) coalesceSize(val int16) int16 {
	if val >= 50 && val <= 200 {
		return val
	}
	return 100
}

func (s *BrandingService) deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
