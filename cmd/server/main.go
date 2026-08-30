package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	commonbranding "github.com/nbt4/cores-common/pkg/branding"
	commonhealth "github.com/nbt4/cores-common/pkg/health"
	"go-barcode-webapp/internal/cache"
	"go-barcode-webapp/internal/compliance"

	"go-barcode-webapp/internal/config"
	"go-barcode-webapp/internal/handlers"
	"go-barcode-webapp/internal/logger"
	"go-barcode-webapp/internal/metrics"
	"go-barcode-webapp/internal/middleware"
	"go-barcode-webapp/internal/models"
	"go-barcode-webapp/internal/monitoring"
	"go-barcode-webapp/internal/repository"
	"go-barcode-webapp/internal/schema"
	"go-barcode-webapp/internal/services"
	pdfsvc "go-barcode-webapp/internal/services/pdf"
	m365sync "go-barcode-webapp/internal/sync/m365"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func buildWarehouseProductsURL(r *http.Request) string {
	warehouseDomain := os.Getenv("WAREHOUSECORE_DOMAIN")

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := warehouseDomain
	if host == "" {
		rawHost := r.Host
		hostname := rawHost
		port := ""

		if h, p, err := net.SplitHostPort(rawHost); err == nil {
			hostname = h
			port = p
		}

		switch {
		case strings.HasPrefix(hostname, "rent."):
			hostname = strings.Replace(hostname, "rent.", "warehouse.", 1)
		case strings.HasPrefix(hostname, "rental."):
			hostname = strings.Replace(hostname, "rental.", "warehouse.", 1)
		case port == "8081":
			hostname = hostname + ":8082"
		case port != "":
			hostname = hostname + ":8082"
		}

		host = hostname
	}

	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s/admin/products", scheme, host)
}

func resolvePackageAliasEndpoint() string {
	aliasURL := strings.TrimSpace(os.Getenv("WAREHOUSECORE_ALIAS_MAP_URL"))
	if aliasURL != "" {
		return strings.TrimRight(aliasURL, "/")
	}

	domain := strings.TrimSpace(os.Getenv("WAREHOUSECORE_DOMAIN"))
	if domain == "" {
		// Fallback to in-compose service name for local/demo setups
		return "http://warehousecore:8082/api/v1/product-packages/alias-map"
	}

	base := strings.TrimRight(domain, "/")
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		return fmt.Sprintf("%s/api/v1/product-packages/alias-map", base)
	}

	scheme := "https"
	lower := strings.ToLower(base)
	// Use http for local/network-internal hosts to avoid TLS failures
	if strings.Contains(base, ":") ||
		strings.HasPrefix(lower, "localhost") ||
		strings.HasPrefix(lower, "warehousecore") ||
		strings.HasSuffix(lower, ".local") ||
		strings.HasPrefix(lower, "127.") ||
		strings.HasPrefix(lower, "10.") ||
		strings.HasPrefix(lower, "192.168.") {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s/api/v1/product-packages/alias-map", scheme, base)
}

func buildWarehouseDevicesURL(r *http.Request) string {
	warehouseDomain := os.Getenv("WAREHOUSECORE_DOMAIN")

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := warehouseDomain
	if host == "" {
		rawHost := r.Host
		hostname := rawHost
		port := ""

		if h, p, err := net.SplitHostPort(rawHost); err == nil {
			hostname = h
			port = p
		}

		switch {
		case strings.HasPrefix(hostname, "rent."):
			hostname = strings.Replace(hostname, "rent.", "warehouse.", 1)
		case strings.HasPrefix(hostname, "rental."):
			hostname = strings.Replace(hostname, "rental.", "warehouse.", 1)
		case port == "8081":
			hostname = hostname + ":8082"
		case port != "":
			hostname = hostname + ":8082"
		}

		host = hostname
	}

	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s/admin/devices", scheme, host)
}

func buildWarehouseCablesURL(r *http.Request) string {
	warehouseDomain := os.Getenv("WAREHOUSECORE_DOMAIN")

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := warehouseDomain
	if host == "" {
		rawHost := r.Host
		hostname := rawHost
		port := ""

		if h, p, err := net.SplitHostPort(rawHost); err == nil {
			hostname = h
			port = p
		}

		switch {
		case strings.HasPrefix(hostname, "rent."):
			hostname = strings.Replace(hostname, "rent.", "warehouse.", 1)
		case strings.HasPrefix(hostname, "rental."):
			hostname = strings.Replace(hostname, "rental.", "warehouse.", 1)
		case port == "8081":
			hostname = hostname + ":8082"
		case port != "":
			hostname = hostname + ":8082"
		}

		host = hostname
	}

	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s/admin/cables", scheme, host)
}

func buildWarehouseCasesURL(r *http.Request) string {
	warehouseDomain := os.Getenv("WAREHOUSECORE_DOMAIN")

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := warehouseDomain
	if host == "" {
		rawHost := r.Host
		hostname := rawHost
		port := ""

		if h, p, err := net.SplitHostPort(rawHost); err == nil {
			hostname = h
			port = p
		}

		switch {
		case strings.HasPrefix(hostname, "rent."):
			hostname = strings.Replace(hostname, "rent.", "warehouse.", 1)
		case strings.HasPrefix(hostname, "rental."):
			hostname = strings.Replace(hostname, "rental.", "warehouse.", 1)
		case port == "8081":
			hostname = hostname + ":8082"
		case port != "":
			hostname = hostname + ":8082"
		}

		host = hostname
	}

	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s/admin/cases", scheme, host)
}

func deprecatedFeatureHandler(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := fmt.Sprintf("%s has been removed from RentalCore.", feature)
		c.AbortWithStatusJSON(http.StatusGone, gin.H{"error": message})
	}
}

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.json", "Configuration file path")
	flag.Parse()

	// Set production mode if environment variable is set
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
		logger.LogInfo("Running in production mode")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.LogInfo("Failed to load config, using defaults: %v", err)
		cfg = &config.Config{}
		cfg.Database.Host = "localhost"
		cfg.Database.Port = 5432
		cfg.Database.Name = "rentalcore"
		cfg.Database.User = "rentalcore"
		cfg.Database.Password = "rentalcore123"
		cfg.Database.SSLMode = "disable"
		cfg.Database.MaxOpenConns = 25
		cfg.Server.Host = "0.0.0.0"
		cfg.Server.Port = 8080
	}

	// Initialize database
	db, err := repository.NewDatabase(&cfg.Database)
	if err != nil {
		logger.LogFatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.LogFatal("Failed to ping database: %v", err)
	}

	// Set initial DB connection gauge
	if sqlDB, err := db.DB.DB(); err == nil {
		metrics.DBConnectionsOpen.Set(float64(sqlDB.Stats().OpenConnections))
	}

	// Ensure M365 sync columns exist on production databases
	if sqlDB, err := db.DB.DB(); err == nil {
		if err := m365sync.EnsureCustomerM365Columns(sqlDB); err != nil {
			logger.LogInfo("Warning: M365 column migration failed: %v", err)
		}
		if err := schema.EnsureVenuesTable(sqlDB); err != nil {
			logger.LogInfo("Warning: venues table migration failed: %v", err)
		}
		if err := schema.EnsureJobsVenueID(sqlDB); err != nil {
			logger.LogInfo("Warning: jobs.venue_id migration failed: %v", err)
		}
	}
	if err := db.DB.AutoMigrate(&models.M365Settings{}); err != nil {
		logger.LogInfo("Warning: M365Settings AutoMigrate failed: %v", err)
	}
	loadSharedM365Settings(db.DB, &cfg.M365)

	// Apply performance indexes for optimal database performance
	go func() {
		if err := config.ApplyPerformanceIndexes(db.DB); err != nil {
			logger.LogInfo("Warning: Failed to apply performance indexes: %v", err)
		} else {
			logger.LogInfo("Performance indexes applied successfully")
		}
	}()

	// Initialize structured logger
	environment := "development"
	if os.Getenv("GIN_MODE") == "release" {
		environment = "production"
	}

	loggerConfig := logger.LoggerConfig{
		Level:        logger.INFO,
		Service:      "rentalcore",
		Version:      "2.1.0",
		Environment:  environment,
		OutputPath:   "", // stdout
		EnableCaller: true,
	}

	if err := logger.InitializeLogger(loggerConfig); err != nil {
		logger.LogFatal("Failed to initialize logger: %v", err)
	}
	defer logger.GlobalLogger.Close()

	// Initialize error tracker
	monitoring.InitializeErrorTracker(1000, 7*24*time.Hour) // 1000 errors, 7 days retention

	// Initialize cache manager
	cacheManager := cache.NewCacheManager()

	// Initialize performance monitor
	perfMonitor := middleware.NewPerformanceMonitor(500 * time.Millisecond) // 500ms slow threshold

	// Initialize compliance system
	complianceMiddleware, err := compliance.NewComplianceMiddleware(
		db.DB,
		"./archives",
		cfg.Security.EncryptionKey,
	)
	if err != nil {
		logger.LogInfo("Warning: Failed to create compliance middleware: %v", err)
		// Use a dummy middleware for development
		complianceMiddleware = nil
	} else {
		// Initialize compliance database tables
		if err := complianceMiddleware.InitializeCompliance(); err != nil {
			logger.LogInfo("Warning: Failed to initialize compliance system: %v", err)
		}
	}

	// Start compliance background tasks
	if complianceMiddleware != nil {
		go complianceMiddleware.PeriodicComplianceCheck(context.Background())
	}

	// Initialize repositories
	jobRepo := repository.NewJobRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	requirementRepo := repository.NewRequirementRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	jobEmployeeRepo := repository.NewJobEmployeeRepository(db.DB)
	skillRepo := repository.NewSkillRepository(db.DB)
	employeeRepo := repository.NewEmployeeRepository(db.DB)
	positionRepo := repository.NewPositionRepository(db)

	// M365 Sync Service (optional — nur wenn Env-Vars gesetzt)
	var m365SyncService handlers.SyncServiceInterface
	var calendarSync handlers.CalendarSyncServiceInterface
	if cfg.M365.IsConfigured() {
		interval, err := time.ParseDuration(cfg.M365.SyncInterval)
		if err != nil {
			interval = 5 * time.Minute
		}
		graphClient := m365sync.NewGraphClient(
			cfg.M365.TenantID,
			cfg.M365.ClientID,
			cfg.M365.ClientSecret,
			cfg.M365.MailboxID,
		)
		exchangeClient := m365sync.NewExchangeAdminClient(
			cfg.M365.TenantID,
			cfg.M365.ClientID,
			cfg.M365.ClientSecret,
		)
		sqlDB, _ := db.DB.DB()
		svc := m365sync.NewSyncService(graphClient, exchangeClient, customerRepo, sqlDB, interval)
		m365SyncService = svc
		syncCtx, _ := context.WithCancel(context.Background())
		svc.Start(syncCtx)
		logger.LogInfo("M365 sync: service initialized")

		calendarClient := m365sync.NewCalendarClient(graphClient, cfg.M365.CalendarMailbox)
		calendarSync = m365sync.NewCalendarSyncService(
			calendarClient, jobRepo, positionRepo, jobEmployeeRepo, db, cfg.M365.AppBaseURL,
		)
		logger.LogInfo("M365 calendar sync: initialized for %s", cfg.M365.CalendarMailbox)
	} else {
		logger.LogInfo("M365 sync: not configured (M365_TENANT_ID etc. not set)")
	}
	statusRepo := repository.NewStatusRepository(db)
	productRepo := repository.NewProductRepository(db)
	jobCategoryRepo := repository.NewJobCategoryRepository(db)
	caseRepo := repository.NewCaseRepository(db)
	equipmentPackageRepo := repository.NewEquipmentPackageRepository(db)
	invoiceRepo := repository.NewInvoiceRepositoryNew(db) // Using NEW fixed repository
	cableRepo := repository.NewCableRepository(db)
	rentalEquipmentRepo := repository.NewRentalEquipmentRepository(db)
	jobAttachmentRepo := repository.NewJobAttachmentRepository(db)
	jobEditSessionRepo := repository.NewJobEditSessionRepository(db)

	// Initialize services
	barcodeService := services.NewBarcodeService()
	jobHistoryService := services.NewJobHistoryService(db.DB)

	// Auto-migration disabled - database schema managed manually
	logger.LogInfo("Database auto-migration disabled - using manual schema management")

	// Initialize job package repository
	jobPackageRepo := repository.NewJobPackageRepository(db)

	// Initialize accessories and consumables repository
	accessoriesConsumablesRepo := repository.NewAccessoriesConsumablesRepository(db)

	// Initialize handlers
	jobHandler := handlers.NewJobHandler(jobRepo, jobPackageRepo, deviceRepo, requirementRepo, customerRepo, statusRepo, jobCategoryRepo, jobEditSessionRepo, jobHistoryService, rentalEquipmentRepo, jobEmployeeRepo, calendarSync)
	jobHistoryHandler := handlers.NewJobHistoryHandler(db.DB)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo, barcodeService, productRepo)
	customerHandler := handlers.NewCustomerHandler(customerRepo, m365SyncService)
	statusHandler := handlers.NewStatusHandler(statusRepo)
	productHandler := handlers.NewProductHandler(productRepo)
	cableHandler := handlers.NewCableHandler(cableRepo)
	infoHandler := handlers.NewInfoHandler()
	barcodeHandler := handlers.NewBarcodeHandler(barcodeService, deviceRepo)
	authHandler := handlers.NewAuthHandler(db.DB, cfg)
	webauthnHandler := handlers.NewWebAuthnHandler(db.DB, cfg)
	profileHandler := handlers.NewProfileHandler(db.DB, cfg, webauthnHandler)
	homeHandler := handlers.NewHomeHandler(jobRepo, deviceRepo, customerRepo, caseRepo, db.DB)

	// Start session cleanup background process
	authHandler.StartSessionCleanup()

	caseHandler := handlers.NewCaseHandler(caseRepo, deviceRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(db.DB)
	searchHandler := handlers.NewSearchHandler(db.DB)
	pwaHandler := handlers.NewPWAHandler(db.DB)
	workflowHandler := handlers.NewWorkflowHandler(jobRepo, customerRepo, equipmentPackageRepo, deviceRepo, db.DB, barcodeService)
	equipmentPackageHandler := handlers.NewEquipmentPackageHandler(equipmentPackageRepo, deviceRepo)
	rentalEquipmentHandler := handlers.NewRentalEquipmentHandler(rentalEquipmentRepo)
	documentHandler, err := handlers.NewDocumentHandler(db.DB)
	if err != nil {
		logger.LogFatal("Failed to create document handler: %v", err)
	}
	financialHandler := handlers.NewFinancialHandler(db.DB)
	securityHandler := handlers.NewSecurityHandler(db.DB)
	invoiceHandler := handlers.NewInvoiceHandlerNew(invoiceRepo, customerRepo, jobRepo, deviceRepo, equipmentPackageRepo, productRepo, &cfg.PDF)
	templateHandler := handlers.NewInvoiceTemplateHandler(invoiceRepo)
	companyProvider := services.NewCompanyProvider(db.DB)
	brandingService := services.NewBrandingService(db.DB, "rental")
	companyHandler := handlers.NewCompanyHandler(db.DB, companyProvider)
	monitoringHandler := handlers.NewMonitoringHandler(db.DB, monitoring.GlobalErrorTracker, perfMonitor, cacheManager)
	jobAttachmentHandler := handlers.NewJobAttachmentHandler(jobAttachmentRepo, jobRepo, jobHistoryService)
	var packageAliasCache *pdfsvc.PackageAliasCache
	if aliasEndpoint := resolvePackageAliasEndpoint(); aliasEndpoint != "" {
		packageAliasCache = pdfsvc.NewPackageAliasCache(aliasEndpoint)
		if packageAliasCache != nil {
			go packageAliasCache.Warm()
			logger.LogInfo("WarehouseCore package alias cache enabled: %s", aliasEndpoint)
		}
	}

	pdfHandler := handlers.NewPDFHandler(db.DB, "uploads", jobHandler, jobAttachmentRepo, packageAliasCache, documentHandler)
	accessoriesConsumablesHandler := handlers.NewAccessoriesConsumablesHandler(accessoriesConsumablesRepo)
	skillHandler := handlers.NewSkillHandler(skillRepo)
	employeeHandler := handlers.NewEmployeeHandler(employeeRepo)
	venueRepo := repository.NewVenueRepository(db)
	venueHandler := handlers.NewVenueHandler(venueRepo)
	positionHandler := handlers.NewPositionHandler(positionRepo, jobRepo, requirementRepo, db.DB)
	m365SettingsHandler := handlers.NewM365SettingsHandler(db.DB)

	// Initialize RBAC middleware for role-based access control
	rbacMiddleware := middleware.NewRBACMiddleware(db.DB)

	// Create default invoice template if none exists
	if err := createDefaultTemplate(templateHandler, invoiceRepo); err != nil {
		logger.LogInfo("Warning: Failed to create default template: %v", err)
	}

	// Setup Gin router with error handling
	r := gin.New()

	// Add monitoring, logging and compliance middleware
	r.Use(logger.GlobalLogger.LoggingMiddleware())
	r.Use(monitoring.GlobalErrorTracker.ErrorTrackingMiddleware())
	r.Use(perfMonitor.PerformanceMiddleware())
	r.Use(middleware.TemplateGlobalsMiddleware()) // Inject cross-navigation URLs

	if complianceMiddleware != nil {
		r.Use(complianceMiddleware.AuditMiddleware())
		r.Use(complianceMiddleware.ComplianceStatusMiddleware())
	}
	r.Use(handlers.GlobalErrorHandler()) // Custom recovery with proper error pages
	r.Use(metrics.Middleware())          // Prometheus metrics

	// Load HTML templates with custom functions
	funcMap := template.FuncMap{
		"deref": func(p *uint) uint {
			if p != nil {
				return *p
			}
			return 0
		},
		"formatDateNew": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 at 3:04 PM")
		},
		"substrNew": func(s string, start, length int) string {
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"derefString": func(p *string) string {
			if p != nil {
				return *p
			}
			return ""
		},
		"derefFloat": func(p *float64) float64 {
			if p != nil {
				return *p
			}
			return 0.0
		},
		"humanizeBytes": func(bytes int64) string {
			if bytes == 0 {
				return "0 B"
			}
			const unit = 1024
			sizes := []string{"B", "KB", "MB", "GB", "TB"}
			i := 0
			for bytes >= unit && i < len(sizes)-1 {
				bytes /= unit
				i++
			}
			if i == 0 {
				return fmt.Sprintf("%d %s", bytes, sizes[i])
			}
			return fmt.Sprintf("%.1f %s", float64(bytes), sizes[i])
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"getStatusColor": func(status string) string {
			switch status {
			case "completed":
				return "success"
			case "pending":
				return "warning"
			case "failed":
				return "danger"
			case "cancelled":
				return "secondary"
			default:
				return "secondary"
			}
		},
		"now": func() time.Time {
			return time.Now()
		},
		"daysAgo": func(date time.Time) int {
			return int(time.Since(date).Hours() / 24)
		},
		"daysUntil": func(date time.Time) int {
			return int(time.Until(date).Hours() / 24)
		},
		"split": func(s, sep string) []string {
			if s == "" {
				return []string{}
			}
			return strings.Split(s, sep)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"timeAgo": func(t *time.Time) string {
			if t == nil {
				return "Never"
			}
			duration := time.Since(*t)
			if duration < time.Minute {
				return "Just now"
			} else if duration < time.Hour {
				minutes := int(duration.Minutes())
				return fmt.Sprintf("%d min ago", minutes)
			} else if duration < 24*time.Hour {
				hours := int(duration.Hours())
				return fmt.Sprintf("%d hours ago", hours)
			} else {
				days := int(duration.Hours() / 24)
				if days == 1 {
					return "Yesterday"
				}
				return fmt.Sprintf("%d days ago", days)
			}
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"substr": func(s string, start, end int) string {
			if len(s) == 0 {
				return ""
			}
			if start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			if start < 0 {
				start = 0
			}
			return s[start:end]
		},
		"mul": func(a, b interface{}) float64 {
			var aVal, bVal float64
			switch v := a.(type) {
			case float64:
				aVal = v
			case *float64:
				if v != nil {
					aVal = *v
				}
			case int:
				aVal = float64(v)
			case uint:
				aVal = float64(v)
			}
			switch v := b.(type) {
			case float64:
				bVal = v
			case *float64:
				if v != nil {
					bVal = *v
				}
			case int:
				bVal = float64(v)
			case uint:
				bVal = float64(v)
			}
			return aVal * bVal
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"len": func(slice interface{}) int {
			switch v := slice.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},
		"hasRole": func(user *models.User, roleName string) bool {
			if user == nil {
				return false
			}
			// System admin always returns true
			if user.Username == "admin" {
				return true
			}
			// Check user roles from database
			var userRoles []models.UserRole
			if err := db.DB.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
				return false
			}
			for _, userRole := range userRoles {
				if userRole.Role != nil && userRole.Role.IsActive && userRole.Role.Name == roleName {
					return true
				}
			}
			return false
		},
		"hasAnyRole": func(user *models.User, roleNames ...string) bool {
			if user == nil {
				return false
			}
			// System admin always returns true
			if user.Username == "admin" {
				return true
			}
			// Check user roles from database
			var userRoles []models.UserRole
			if err := db.DB.Preload("Role").Where("userID = ? AND is_active = ?", user.UserID, true).Find(&userRoles).Error; err != nil {
				return false
			}
			for _, userRole := range userRoles {
				if userRole.Role == nil || !userRole.Role.IsActive {
					continue
				}
				for _, roleName := range roleNames {
					if userRole.Role.Name == roleName {
						return true
					}
				}
			}
			return false
		},
		"companyName": func() string {
			return companyProvider.CompanyName()
		},
		"brandingLogo": func(position string) string {
			branding := brandingService.GetConfig()
			switch position {
			case "mark":
				if branding.Assets.MarkOnDark != "" {
					return branding.Assets.MarkOnDark
				}
				return "/static/images/logos/rentalcore_white_icon.svg"
			case "favicon":
				if branding.FaviconPath != "" {
					return branding.FaviconPath
				}
				return "/static/images/logos/rentalcore_black_icon.svg"
			case "stacked":
				if branding.Assets.StackedOnDark != "" {
					return branding.Assets.StackedOnDark
				}
				return "/static/images/logos/rentalcore_white_full.svg"
			default:
				if branding.Assets.HorizontalOnDark != "" {
					return branding.Assets.HorizontalOnDark
				}
				return "/static/images/logos/rentalcore_white_side.svg"
			}
		},
	}
	r.SetFuncMap(funcMap)
	// Load HTML templates only if they exist (legacy support)
	if _, err := os.Stat("web/templates"); err == nil {
		r.LoadHTMLGlob("web/templates/*")
	}

	// Health check endpoint (no auth required)
	sqlDB, _ := db.DB.DB()
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health", gin.WrapH(commonhealth.Handler(sqlDB, "rentalcore", "5.3.95")))
	r.GET("/api/v1/branding", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusOK, brandingService.GetConfig())
	})
	r.GET("/logos/:filename", func(c *gin.Context) {
		filename := filepath.Base(c.Param("filename"))
		if filename == "." || filename == "" || filename != c.Param("filename") {
			c.Status(http.StatusNotFound)
			return
		}
		path := filepath.Join("/var/lib/branding/logos", filename)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.Header("X-Content-Type-Options", "nosniff")
			c.File(path)
			return
		}
		c.Status(http.StatusNotFound)
	})

	// Serve React SPA build assets
	r.StaticFS("/assets", http.Dir("web/dist/assets"))
	// The same build is mounted below /rentalcore/ by Cores Dashboard. Keep the
	// asset alias on the dedicated RentalCore domain so one artifact serves both.
	r.StaticFS("/rentalcore/assets", http.Dir("web/dist/assets"))

	// Add caching for static files
	r.StaticFS("/static", http.Dir("web/static"))
	r.StaticFS("/rentalcore/static", http.Dir("web/static"))
	r.StaticFS("/uploads", http.Dir("uploads"))
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("ETag", fmt.Sprintf(`"%x"`, time.Now().Unix()))
		}
		c.Next()
	})

	// PWA Service Worker route
	r.GET("/sw.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File("web/static/sw.js")
	})

	// PWA public routes (no authentication required)
	manifestHandler := func(c *gin.Context) {
		c.Header("Content-Type", "application/manifest+json")
		c.Header("Cache-Control", "no-cache")
		mountPath := strings.TrimSuffix(c.GetHeader("X-Forwarded-Prefix"), "/")
		if mountPath != "/rentalcore" {
			mountPath = ""
		}
		mountedPath := func(value string) string { return mountPath + value }
		c.JSON(http.StatusOK, commonbranding.Manifest(brandingService.GetConfig(), commonbranding.ManifestOptions{
			Name: "RentalCore", StartURL: mountedPath("/"), Scope: mountedPath("/"),
			FallbackIcon192:  mountedPath("/static/images/app-icons/icon-192.png"),
			FallbackIcon512:  mountedPath("/static/images/app-icons/icon-512.png"),
			FallbackMaskable: mountedPath("/static/images/app-icons/icon-maskable-512.png"),
		}))
	}
	r.GET("/manifest.json", manifestHandler)
	r.GET("/rentalcore/manifest.json", manifestHandler)

	// Favicon route
	r.GET("/favicon.ico", func(c *gin.Context) {
		cfg := brandingService.GetConfig()
		c.Header("Cache-Control", "no-cache")
		if cfg.FaviconPath != "" {
			c.Redirect(http.StatusTemporaryRedirect, cfg.FaviconPath)
			return
		}
		c.File("web/static/images/logos/rentalcore_black_icon.png")
	})

	// Initialize default roles
	if err := securityHandler.InitializeDefaultRoles(); err != nil {
		logger.LogInfo("Failed to initialize default roles: %v", err)
	}

	// Routes
	setupRoutes(r, cfg, jobHandler, jobHistoryHandler, deviceHandler, customerHandler, statusHandler, productHandler, cableHandler, infoHandler, barcodeHandler, authHandler, webauthnHandler, homeHandler, profileHandler, caseHandler, analyticsHandler, searchHandler, pwaHandler, workflowHandler, equipmentPackageHandler, rentalEquipmentHandler, documentHandler, financialHandler, securityHandler, invoiceHandler, templateHandler, companyHandler, monitoringHandler, jobAttachmentHandler, pdfHandler, accessoriesConsumablesHandler, positionHandler, rbacMiddleware, complianceMiddleware, skillHandler, employeeHandler, db, venueHandler, m365SettingsHandler)

	// Add dedicated error route
	r.GET("/error", func(c *gin.Context) {
		code := c.DefaultQuery("code", "500")
		message := c.DefaultQuery("message", "Internal Server Error")
		details := c.DefaultQuery("details", "Something went wrong on the server")

		codeInt, _ := strconv.Atoi(code)
		c.JSON(codeInt, gin.H{"error": message, "details": details})
	})

	// SPA fallback: serve React index.html for all non-API, non-static routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API and legacy routes return JSON 404
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/assets/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Everything else: serve the SPA shell
		c.File("web/dist/index.html")
	})

	// Start server
	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	// Wrap Gin router with method override support
	methodOverrideHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		originalMethod := req.Method
		if req.Method == "POST" {
			contentType := req.Header.Get("Content-Type")
			logger.LogInfo("POST request to %s with Content-Type: '%s'", req.URL.Path, contentType)

			if contentType == "application/x-www-form-urlencoded" ||
				strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {

				if err := req.ParseForm(); err == nil {
					methodParam := req.FormValue("_method")
					logger.LogInfo("Form _method parameter: '%s'", methodParam)
					if methodParam == "PUT" || methodParam == "DELETE" {
						logger.LogInfo("Method override: %s -> %s for path: %s", originalMethod, methodParam, req.URL.Path)
						req.Method = methodParam
					}
				}
			}
		}
		// Pass to Gin router
		r.ServeHTTP(w, req)
	})

	zlog := logger.GlobalLogger.Raw()
	zlog.Info().Str("addr", addr).Msg("Server starting")

	srv := &http.Server{
		Addr:    addr,
		Handler: methodOverrideHandler,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Fatal().Err(err).Msg("Server failed")
		}
	}()

	<-ctx.Done()
	zlog.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Error().Err(err).Msg("Server forced to shutdown")
	}

	zlog.Info().Msg("Server exited gracefully")
}

// loadSharedM365Settings makes the Cores dashboard's single Microsoft app
// registration authoritative for RentalCore's existing contacts/calendar sync.
// Environment/config values remain a fallback for installations that have not
// configured the central dashboard yet.
func loadSharedM365Settings(db *gorm.DB, target *config.M365Config) {
	var settings models.M365Settings
	if err := db.First(&settings, 1).Error; err != nil {
		logger.LogInfo("M365 settings: central dashboard configuration not available, using environment fallback")
		return
	}
	if settings.TenantID != "" {
		target.TenantID = settings.TenantID
	}
	if settings.ClientID != "" {
		target.ClientID = settings.ClientID
	}
	if settings.ClientSecret != "" {
		target.ClientSecret = settings.ClientSecret
	}
	if settings.MailboxID != "" {
		target.MailboxID = settings.MailboxID
	}
	if settings.SyncInterval != "" {
		target.SyncInterval = settings.SyncInterval
	}
	if settings.CalendarMailbox != "" {
		target.CalendarMailbox = settings.CalendarMailbox
	}
	if settings.AppBaseURL != "" {
		target.AppBaseURL = settings.AppBaseURL
	}
	logger.LogInfo("M365 settings: loaded shared app registration from Cores dashboard")
}

func setupRoutes(r *gin.Engine,
	cfg *config.Config,
	jobHandler *handlers.JobHandler,
	jobHistoryHandler *handlers.JobHistoryHandler,
	deviceHandler *handlers.DeviceHandler,
	customerHandler *handlers.CustomerHandler,
	statusHandler *handlers.StatusHandler,
	productHandler *handlers.ProductHandler,
	cableHandler *handlers.CableHandler,
	infoHandler *handlers.InfoHandler,
	barcodeHandler *handlers.BarcodeHandler,
	authHandler *handlers.AuthHandler,
	webauthnHandler *handlers.WebAuthnHandler,
	homeHandler *handlers.HomeHandler,
	profileHandler *handlers.ProfileHandler,
	caseHandler *handlers.CaseHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	searchHandler *handlers.SearchHandler,
	pwaHandler *handlers.PWAHandler,
	workflowHandler *handlers.WorkflowHandler,
	equipmentPackageHandler *handlers.EquipmentPackageHandler,
	rentalEquipmentHandler *handlers.RentalEquipmentHandler,
	documentHandler *handlers.DocumentHandler,
	financialHandler *handlers.FinancialHandler,
	securityHandler *handlers.SecurityHandler,
	invoiceHandler *handlers.InvoiceHandlerNew,
	templateHandler *handlers.InvoiceTemplateHandler,
	companyHandler *handlers.CompanyHandler,
	monitoringHandler *handlers.MonitoringHandler,
	jobAttachmentHandler *handlers.JobAttachmentHandler,
	pdfHandler *handlers.PDFHandler,
	accessoriesConsumablesHandler *handlers.AccessoriesConsumablesHandler,
	positionHandler *handlers.PositionHandler,
	rbacMiddleware *middleware.RBACMiddleware,
	complianceMiddleware *compliance.ComplianceMiddleware,
	skillHandler *handlers.SkillHandler,
	employeeHandler *handlers.EmployeeHandler,
	db *repository.Database,
	venueHandler *handlers.VenueHandler,
	m365SettingsHandler *handlers.M365SettingsHandler) {

	// Root route - serve SPA
	r.GET("/", func(c *gin.Context) {
		c.File("web/dist/index.html")
	})

	// SPA routes - serve index.html (React Router handles client-side routing)
	r.GET("/login", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.GET("/change-password", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.GET("/logout", func(c *gin.Context) {
		// Clear session cookie and redirect to SPA
		c.SetCookie("session_id", "", -1, "/", "", false, true)
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	})

	// JSON API auth endpoints for React frontend (no session required)
	apiAuth := r.Group("/api/v1/auth")
	{
		apiAuth.POST("/login", authHandler.LoginAPI)
		apiAuth.POST("/logout", authHandler.LogoutAPI)
		apiAuth.GET("/me", authHandler.AuthMiddleware(), authHandler.MeAPI)
		apiAuth.POST("/change-password", authHandler.AuthMiddleware(), authHandler.ChangePasswordAPI)
	}

	// Passkey authentication routes (no auth required for login)
	auth := r.Group("/auth")
	{
		// Force password change - serve SPA (React handles the flow)
		auth.GET("/force-password-change", func(c *gin.Context) { c.File("web/dist/index.html") })
		auth.POST("/force-password-change", authHandler.HandleForcePasswordChange)
	}

	// Additional auth routes
	auth = r.Group("/auth")
	{
		passkey := auth.Group("/passkey")
		{
			passkey.POST("/start-authentication", webauthnHandler.StartPasskeyAuthentication)
			passkey.POST("/complete-authentication", webauthnHandler.CompletePasskeyAuthentication)
		}
	}

	// Protected routes - require authentication
	protected := r.Group("/")
	protected.Use(authHandler.AuthMiddleware())
	{
		// SPA page routes - serve React frontend
		spaHandler := func(c *gin.Context) { c.File("web/dist/index.html") }
		protected.GET("/dashboard", spaHandler)
		protected.GET("/help", spaHandler)
		protected.GET("/about", spaHandler)
		protected.GET("/contact", spaHandler)
		protected.GET("/admin", spaHandler)

		// Job routes
		jobs := protected.Group("/jobs")
		{
			jobs.GET("", spaHandler)
			jobs.GET("/new", spaHandler)
			jobs.GET("/new/", spaHandler)
			jobs.POST("", jobHandler.CreateJob)
			jobs.GET("/:id", spaHandler)
			jobs.GET("/:id/edit", spaHandler)
			jobs.PUT("/:id", jobHandler.UpdateJob)
			jobs.POST("/:id/update", jobHandler.UpdateJob)
			jobs.DELETE("/:id", jobHandler.DeleteJob)
			jobs.GET("/:id/devices", jobHandler.GetJobDevices)
			jobs.POST("/:id/devices", jobHandler.AssignDevice)
			jobs.DELETE("/:id/devices/:deviceId", jobHandler.RemoveDevice)
		}

		// Device routes - redirect to WarehouseCore
		redirectToWarehouseDevices := func(c *gin.Context) {
			target := buildWarehouseDevicesURL(c.Request)
			if target == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "WarehouseCore domain not configured",
					"message": "Set WAREHOUSECORE_DOMAIN to enable device management in WarehouseCore.",
				})
				return
			}
			c.Redirect(http.StatusFound, target)
		}

		devices := protected.Group("/devices")
		{
			// Redirect web UI to WarehouseCore
			devices.GET("", redirectToWarehouseDevices)
			devices.GET("/new", redirectToWarehouseDevices)
			devices.GET("/:id/edit", redirectToWarehouseDevices)

			// Keep read-only endpoints for Jobs/Invoices integration
			devices.GET("/:id", deviceHandler.GetDevice)
			devices.GET("/:id/stats", deviceHandler.GetDeviceStatsAPI)
			devices.GET("/available", deviceHandler.GetAvailableDevices)

			// QR/Barcode redirected to WarehouseCore
			devices.GET("/:id/qr", redirectToWarehouseDevices)
			devices.GET("/:id/barcode", redirectToWarehouseDevices)

			// Write operations removed - handled in WarehouseCore
			// POST, PUT, DELETE routes removed
		}

		redirectToWarehouseProducts := func(c *gin.Context) {
			target := buildWarehouseProductsURL(c.Request)
			if target == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "WarehouseCore domain not configured",
					"message": "Set WAREHOUSECORE_DOMAIN to enable product management in WarehouseCore.",
				})
				return
			}
			c.Redirect(http.StatusFound, target)
		}

		protected.GET("/products", redirectToWarehouseProducts)
		protected.GET("/products/new", redirectToWarehouseProducts)

		// Rental Equipment routes
		rentalEquipment := protected.Group("/rental-equipment")
		{
			rentalEquipment.GET("", spaHandler)
			rentalEquipment.GET("/new", spaHandler)
			rentalEquipment.GET("/:id/edit", spaHandler)
			rentalEquipment.GET("/analytics", spaHandler)
		}

		// Cable routes - redirect to WarehouseCore
		redirectToWarehouseCables := func(c *gin.Context) {
			target := buildWarehouseCablesURL(c.Request)
			if target == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WarehouseCore domain not configured"})
				return
			}
			c.Redirect(http.StatusTemporaryRedirect, target)
		}

		cables := protected.Group("/cables")
		{
			// Redirect to WarehouseCore
			cables.GET("", redirectToWarehouseCables)
			cables.GET("/new", redirectToWarehouseCables)
		}

		// Customer routes
		customers := protected.Group("/customers")
		{
			customers.GET("", spaHandler)
			customers.GET("/new", spaHandler)
			customers.POST("", customerHandler.CreateCustomer)
			customers.GET("/:id", spaHandler)
			customers.GET("/:id/edit", spaHandler)
			customers.PUT("/:id", customerHandler.UpdateCustomer)
			customers.DELETE("/:id", customerHandler.DeleteCustomer)
		}

		// Status routes
		statuses := protected.Group("/statuses")
		{
			statuses.GET("", statusHandler.ListStatuses)
		}

		// Case routes - redirect to WarehouseCore
		redirectToWarehouseCases := func(c *gin.Context) {
			target := buildWarehouseCasesURL(c.Request)
			if target == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "WarehouseCore domain not configured",
					"message": "Set WAREHOUSECORE_DOMAIN to enable case management in WarehouseCore.",
				})
				return
			}
			c.Redirect(http.StatusFound, target)
		}

		cases := protected.Group("/cases")
		{
			// Redirect to WarehouseCore
			cases.GET("", redirectToWarehouseCases)
			cases.GET("/new", redirectToWarehouseCases)
			cases.GET("/:id/edit", redirectToWarehouseCases)
			cases.GET("/:id/devices", redirectToWarehouseCases)

			// Keep read-only endpoint for Jobs integration (if needed)
			cases.GET("/:id", caseHandler.GetCase)
		}

		// Barcode routes
		barcodes := protected.Group("/barcodes")
		{
			barcodes.GET("/device/:serialNo/qr", barcodeHandler.GenerateDeviceQR)
			barcodes.GET("/device/:serialNo/barcode", barcodeHandler.GenerateDeviceBarcode)
		}

		// Analytics routes
		analytics := protected.Group("/analytics")
		{
			analytics.GET("", spaHandler)
			analytics.GET("/revenue", analyticsHandler.GetRevenueAPI)
			analytics.GET("/revenue/drilldown", analyticsHandler.GetRevenueDrilldown)
			analytics.GET("/equipment", analyticsHandler.GetEquipmentAPI)
			analytics.GET("/devices/all", analyticsHandler.GetAllDeviceRevenuesAPI)
			analytics.GET("/devices/:deviceId", analyticsHandler.GetDeviceAnalytics)
			analytics.GET("/export", analyticsHandler.ExportAnalytics)
		}

		// Search routes
		search := protected.Group("/search")
		{
			search.GET("/global", searchHandler.GlobalSearch)
			search.POST("/advanced", searchHandler.AdvancedSearch)
			search.GET("/suggestions", searchHandler.SearchSuggestions)
			search.GET("/saved", searchHandler.SavedSearches)
			search.DELETE("/saved/:id", searchHandler.DeleteSavedSearch)
		}

		// PDF UI routes
		pdfUI := protected.Group("/pdf")
		{
			pdfUI.GET("/review/:upload_id", pdfHandler.ShowReviewScreen)
			pdfUI.GET("/mapping/:extraction_id", pdfHandler.ShowMappingScreen)
		}

		// PWA routes
		pwa := protected.Group("/pwa")
		{
			pwa.POST("/subscribe", pwaHandler.SubscribePush)
			pwa.POST("/unsubscribe", pwaHandler.UnsubscribePush)
			pwa.POST("/sync", pwaHandler.SyncOfflineData)
			pwa.GET("/manifest", pwaHandler.GetOfflineManifest)
			pwa.GET("/install", pwaHandler.InstallPrompt)
			pwa.GET("/status", pwaHandler.GetConnectionStatus)
		}

		// Workflow routes
		workflow := protected.Group("/workflow")
		{
			// Equipment Packages
			packagesDisabled := deprecatedFeatureHandler("Equipment packages")
			workflow.Any("/packages", packagesDisabled)
			workflow.Any("/packages/*path", packagesDisabled)

			// Bulk Operations
			bulk := workflow.Group("/bulk")
			{
				bulk.GET("", workflowHandler.BulkOperationsForm)
				bulk.POST("/update-status", workflowHandler.BulkUpdateDeviceStatus)
				bulk.POST("/assign-job", workflowHandler.BulkAssignToJob)
				bulk.POST("/generate-qr", workflowHandler.BulkGenerateQRCodes)
			}

			// Workflow API
			workflow.GET("/stats", workflowHandler.GetWorkflowStats)
		}

		// Document routes
		documents := protected.Group("/documents")
		{
			documents.GET("", spaHandler)
			documents.GET("/upload", documentHandler.UploadDocumentForm)
			documents.POST("/upload", documentHandler.UploadDocument)
			pool := documents.Group("/pool")
			pool.Use(rbacMiddleware.RequireAdminOrManager())
			pool.GET("", documentHandler.ListFilePool)
			pool.GET("/sync", documentHandler.SyncFilePool)
			documents.GET("/:id", documentHandler.GetDocument)
			documents.GET("/:id/view", documentHandler.ViewDocument)
			documents.GET("/:id/download", documentHandler.DownloadDocument)
			documents.DELETE("/:id", documentHandler.DeleteDocument)
			documents.GET("/:id/sign", documentHandler.SignatureForm)
			documents.POST("/:id/sign", documentHandler.AddSignature)
			documents.GET("/signatures/:id/verify", documentHandler.VerifySignature)
		}

		// Financial routes
		financial := protected.Group("/financial")
		{
			financial.GET("", financialHandler.FinancialDashboard)
			financial.GET("/transactions", financialHandler.ListTransactions)
			financial.GET("/transactions/new", financialHandler.NewTransactionForm)
			financial.POST("/transactions", financialHandler.CreateTransaction)
			financial.GET("/transactions/:id", financialHandler.GetTransaction)
			financial.PUT("/transactions/:id/status", financialHandler.UpdateTransactionStatus)
			financial.POST("/jobs/:jobId/invoice", financialHandler.GenerateInvoice)
			financial.GET("/reports", financialHandler.FinancialReports)
			financial.GET("/api/revenue-report", financialHandler.GetRevenueReport)
			financial.GET("/api/payment-report", financialHandler.GetPaymentReport)

			// Export routes
			financial.GET("/api/export/transactions", financialHandler.ExportTransactions)
			financial.GET("/api/export/revenue", financialHandler.ExportRevenue)
			financial.GET("/api/export/tax-report", financialHandler.ExportTaxReportCSV)
		}

		// Invoice routes (using NEW fixed invoice system with GoBD compliance)
		invoices := protected.Group("/invoices")
		if complianceMiddleware != nil {
			invoices.Use(complianceMiddleware.InvoiceComplianceMiddleware())
			invoices.Use(complianceMiddleware.DataProcessingMiddleware(compliance.FinancialData, "invoice_management", "Contract performance and accounting", "Art. 6(1)(b) GDPR"))
		}
		{
			invoicesDisabled := deprecatedFeatureHandler("Invoices")
			invoices.Any("", invoicesDisabled)
			invoices.Any("/*path", invoicesDisabled)
		}

		// Invoice template routes - full implementation
		invoiceTemplates := protected.Group("/invoice-templates")
		{
			templatesDisabled := deprecatedFeatureHandler("Invoice templates")
			invoiceTemplates.Any("", templatesDisabled)
			invoiceTemplates.Any("/*path", templatesDisabled)
		}

		// Company Settings routes - NOW ACTIVE
		settings := protected.Group("/settings")
		{
			// Company settings form route
			settings.GET("/company", func(c *gin.Context) {
				companyHandler.CompanySettingsForm(c)
			})
			settings.POST("/company", companyHandler.UpdateCompanySettingsForm)
			settings.GET("/company/api", companyHandler.GetCompanySettings)
			settings.PUT("/company/api", companyHandler.UpdateCompanySettings)
			settings.POST("/company/logo", companyHandler.UploadCompanyLogo)
			settings.DELETE("/company/logo", companyHandler.DeleteCompanyLogo)
			settings.GET("/mappings", pdfHandler.ShowMappingManagement)

		}

		// Security & Admin routes (PROTECTED - Admin only)
		security := protected.Group("/security")
		security.Use(rbacMiddleware.RequireAdmin()) // Require admin role for all security routes
		{
			// Web interface routes
			security.GET("/roles", func(c *gin.Context) {
				c.Redirect(http.StatusTemporaryRedirect, "/admin")
			})
			security.GET("/audit", securityHandler.SecurityAuditPage)

			// Role management API
			rolesAPI := security.Group("/api/roles")
			{
				rolesAPI.GET("", securityHandler.GetRoles)
				rolesAPI.GET("/:id", securityHandler.GetRole)
				rolesAPI.POST("", securityHandler.CreateRole)
				rolesAPI.PUT("/:id", securityHandler.UpdateRole)
				rolesAPI.DELETE("/:id", securityHandler.DeleteRole)
			}

			// User role management API
			userRolesAPI := security.Group("/api/users")
			{
				userRolesAPI.GET("/:userId/roles", securityHandler.GetUserRoles)
				userRolesAPI.POST("/:userId/roles", securityHandler.AssignUserRole)
				userRolesAPI.DELETE("/:userId/roles/:roleId", securityHandler.RevokeUserRole)
			}

			// Admin user management API
			adminAPI := security.Group("/api/admin")
			{
				adminAPI.PUT("/users/:id/password", authHandler.AdminSetUserPassword)
				adminAPI.PUT("/users/:id/status", authHandler.AdminBlockUser)
			}

			// Audit API
			auditAPI := security.Group("/api/audit")
			{
				auditAPI.GET("", securityHandler.GetAuditLogs)
				auditAPI.GET("/:id", securityHandler.GetAuditLog)
				auditAPI.GET("/export", securityHandler.ExportAuditLogs)
			}

			// Permissions API
			permissionsAPI := security.Group("/api/permissions")
			{
				permissionsAPI.GET("", securityHandler.GetPermissions)
				permissionsAPI.GET("/definitions", securityHandler.GetPermissionDefinitionsAPI)
				permissionsAPI.GET("/check", securityHandler.CheckPermission)
			}
		}

		// System Monitoring routes (PROTECTED - Admin only)
		monitoring := protected.Group("/monitoring")
		monitoring.Use(rbacMiddleware.RequireAdmin()) // Require admin role for monitoring
		{
			monitoring.GET("", func(c *gin.Context) {
				monitoringHandler.Dashboard(c)
			})
			monitoring.GET("/health", monitoringHandler.GetApplicationHealth)
			monitoring.GET("/metrics", monitoringHandler.GetSystemMetrics)
			monitoring.GET("/metrics/prometheus", monitoringHandler.ExportMetrics)
			monitoring.GET("/performance", monitoringHandler.GetPerformanceMetrics)
			monitoring.GET("/errors", monitoringHandler.GetErrorDetails)
			monitoring.POST("/errors/:fingerprint/resolve", monitoringHandler.ResolveError)
			monitoring.POST("/test-error", monitoringHandler.TriggerTestError)
			monitoring.GET("/logs", monitoringHandler.GetLogStream)
		}

		// Compliance routes (GoBD & GDPR)
		if complianceMiddleware != nil {
			compliance := protected.Group("/compliance")
			{
				compliance.GET("/status", complianceMiddleware.GetComplianceStatus())
				compliance.POST("/retention/cleanup", complianceMiddleware.RetentionCleanupMiddleware())
				compliance.POST("/gdpr/request", complianceMiddleware.GDPRRequestMiddleware())
			}
		}

		// Profile Settings routes (moved to end to avoid potential conflicts)
		profile := protected.Group("/profile")
		{
			profile.GET("/security-status", profileHandler.SecurityStatus)

			// WebAuthn (Passkey) routes
			passkeys := profile.Group("/passkeys")
			{
				passkeys.POST("/start-registration", profileHandler.StartPasskeyRegistration)
				passkeys.POST("/complete-registration", profileHandler.CompletePasskeyRegistration)
				passkeys.GET("", profileHandler.ListUserPasskeys)
				passkeys.DELETE("/:id", profileHandler.DeletePasskey)
			}

			// 2FA routes
			twoFA := profile.Group("/2fa")
			{
				twoFA.POST("/setup", profileHandler.Setup2FA)
				twoFA.POST("/verify", profileHandler.Verify2FA)
				twoFA.POST("/disable", profileHandler.Disable2FA)
				twoFA.GET("/status", profileHandler.Get2FAStatus)
			}
		}

		// User Management (PROTECTED - Admin/Manager only)
		userManagement := protected.Group("")
		userManagement.Use(rbacMiddleware.RequireAdminOrManager()) // Require admin or manager role
		{
			// Main user management routes
			userManagement.GET("/users", authHandler.ListUsers)
			userManagement.POST("/users", authHandler.CreateUserWeb)
			userManagement.PUT("/users/:id/password", authHandler.AdminSetUserPassword)

			// User form and management routes with no parameter conflicts
			userManagement.GET("/user-management/new", authHandler.NewUserForm)
			userManagement.GET("/user-management/:id/edit", authHandler.EditUserForm)
			userManagement.GET("/user-management/:id/view", authHandler.GetUser)
			userManagement.PUT("/user-management/:id", authHandler.UpdateUser)
			userManagement.DELETE("/user-management/:id", authHandler.DeleteUser)

			// Direct explicit routes for old paths - NO parameter routes under /users
			userManagement.GET("/users/new", func(c *gin.Context) {
				c.Redirect(http.StatusSeeOther, "/user-management/new")
			})
		}

		// API routes
		api := protected.Group("/api/v1")
		{
			api.GET("/analytics/revenue/drilldown", analyticsHandler.GetRevenueDrilldown)

			// Statuses API
			api.GET("/statuses", statusHandler.ListStatuses)

			// Job API
			apiJobs := api.Group("/jobs")
			{
				apiJobs.GET("", jobHandler.ListJobsAPI)
				apiJobs.POST("", jobHandler.CreateJobAPI)
				apiJobs.GET("/:id", jobHandler.GetJobAPI)
				apiJobs.PUT("/:id", jobHandler.UpdateJobAPI)
				apiJobs.DELETE("/:id", jobHandler.DeleteJobAPI)
				apiJobs.GET("/:id/devices", jobHandler.GetJobDevices)
				apiJobs.POST("/:id/devices/:deviceId", jobHandler.AssignDeviceAPI)
				apiJobs.PUT("/:id/devices/:deviceId", jobHandler.UpdateDevicePriceAPI)
				apiJobs.DELETE("/:id/devices/:deviceId", jobHandler.RemoveDeviceAPI)
				apiJobs.GET("/:id/requirements", jobHandler.GetJobRequirementsAPI)
				apiJobs.GET("/:id/products/:product_id/available-devices", jobHandler.GetAvailableDevicesForRequirementAPI)
				apiJobs.POST("/:id/editing", jobHandler.StartJobEditingSession)
				apiJobs.DELETE("/:id/editing", jobHandler.StopJobEditingSession)
				apiJobs.GET("/:id/editing", jobHandler.GetJobEditingSessions)
				apiJobs.GET("/:id/history", jobHistoryHandler.GetJobHistory)
				apiJobs.GET("/:id/cable-planning", jobHandler.GetJobCablePlanning)

				// Job package routes
				apiJobs.GET("/:id/packages", jobHandler.GetJobPackages)
				apiJobs.POST("/:id/packages", jobHandler.AssignPackageToJob)

				// Job Attachments routes (matching frontend expectations for /api/v1/jobs/...)
				apiJobs.GET("/:id/attachments", jobAttachmentHandler.GetJobAttachments)
				apiJobs.POST("/attachments/upload", jobAttachmentHandler.UploadAttachment)
				apiJobs.GET("/attachments/:id/view", jobAttachmentHandler.ViewAttachment)
				apiJobs.GET("/attachments/:id/download", jobAttachmentHandler.DownloadAttachment)
				apiJobs.DELETE("/attachments/:id", jobAttachmentHandler.DeleteAttachment)
				apiJobs.PUT("/attachments/:id/description", jobAttachmentHandler.UpdateAttachmentDescription)
			}

			// Job position routes
			apiJobs.GET("/:id/positions", positionHandler.GetPositions)
			apiJobs.POST("/:id/positions", positionHandler.CreatePosition)
			apiJobs.PUT("/:id/positions/:posId", positionHandler.UpdatePosition)
			apiJobs.DELETE("/:id/positions/:posId", positionHandler.DeletePosition)
			apiJobs.PATCH("/:id/positions/reorder", positionHandler.ReorderPositions)
			apiJobs.POST("/:id/positions/:posId/devices", positionHandler.AssignDevice)
			apiJobs.DELETE("/:id/positions/:posId/devices/:devId", positionHandler.RemoveDevice)
			apiJobs.GET("/:id/totals", positionHandler.GetTotals)
			apiJobs.PATCH("/:id/price-settings", positionHandler.UpdatePriceSettings)
			apiJobs.GET("/:id/picklist", positionHandler.GetPicklist)

			// Rental catalog for job position selection
			api.GET("/rental-catalog", positionHandler.GetRentalCatalog)

			// Service items (used by JobPositionsPanel)
			api.GET("/service-items", pdfHandler.ListServiceItems)

			// Job package management routes
			apiJobPackages := api.Group("/jobs/packages")
			{
				apiJobPackages.PATCH("/:package_id/price", jobHandler.UpdateJobPackagePrice)
				apiJobPackages.PATCH("/:package_id/quantity", jobHandler.UpdateJobPackageQuantity)
				apiJobPackages.DELETE("/:package_id", jobHandler.RemoveJobPackage)
				apiJobPackages.GET("/:package_id/reservations", jobHandler.GetJobPackageReservations)
			}

			// Device API
			apiDevices := api.Group("/devices")
			{
				apiDevices.GET("", deviceHandler.ListDevicesAPI)
				warehouseWriteRedirect := func(c *gin.Context) {
					target := buildWarehouseDevicesURL(c.Request)
					message := gin.H{
						"error":   "Device write APIs moved to WarehouseCore",
						"message": "Use WarehouseCore for creating, updating, or deleting devices.",
					}
					if target != "" {
						message["redirect"] = target
					}
					c.JSON(http.StatusGone, message)
				}
				apiDevices.POST("", warehouseWriteRedirect)
				apiDevices.GET("/available", deviceHandler.GetAvailableDevicesAPI)
				apiDevices.GET("/available/job/:jobId", deviceHandler.GetAvailableDevicesForJobAPI)
				apiDevices.GET("/tree/availability", deviceHandler.GetDeviceTreeWithAvailability)
				apiDevices.GET("/:id", deviceHandler.GetDeviceAPI)
				apiDevices.PUT("/:id", warehouseWriteRedirect)
				apiDevices.DELETE("/:id", warehouseWriteRedirect)
			}

			// Product API
			apiProducts := api.Group("/products")
			{
				apiProducts.GET("", productHandler.ListProducts)
				apiProducts.GET("/:id", productHandler.GetProductAPI)

				// Product Dependencies (WarehouseCore integration)
				apiProducts.GET("/:id/dependencies", accessoriesConsumablesHandler.GetProductDependenciesAPI)

				// Product Accessories
				apiProducts.GET("/:id/accessories", accessoriesConsumablesHandler.GetProductAccessoriesAPI)
				apiProducts.POST("/:id/accessories", accessoriesConsumablesHandler.AddProductAccessoryAPI)
				apiProducts.DELETE("/:id/accessories/:accessoryID", accessoriesConsumablesHandler.RemoveProductAccessoryAPI)

				// Product Consumables
				apiProducts.GET("/:id/consumables", accessoriesConsumablesHandler.GetProductConsumablesAPI)
				apiProducts.POST("/:id/consumables", accessoriesConsumablesHandler.AddProductConsumableAPI)
				apiProducts.DELETE("/:id/consumables/:consumableID", accessoriesConsumablesHandler.RemoveProductConsumableAPI)
			}

			// Accessories & Consumables API
			apiAccessories := api.Group("/accessories")
			{
				apiAccessories.GET("/products", accessoriesConsumablesHandler.GetAccessoryProductsAPI)
			}

			apiConsumables := api.Group("/consumables")
			{
				apiConsumables.GET("/products", accessoriesConsumablesHandler.GetConsumableProductsAPI)
			}

			// Count Types API
			apiCountTypes := api.Group("/count-types")
			{
				apiCountTypes.GET("", accessoriesConsumablesHandler.GetCountTypesAPI)
			}

			// Job Accessories API
			apiJobAccessories := api.Group("/jobs")
			{
				apiJobAccessories.GET("/:id/accessories", accessoriesConsumablesHandler.GetJobAccessoriesAPI)
				apiJobAccessories.POST("/:id/accessories", accessoriesConsumablesHandler.CreateJobAccessoryAPI)
				apiJobAccessories.PUT("/accessories/:id", accessoriesConsumablesHandler.UpdateJobAccessoryAPI)
				apiJobAccessories.DELETE("/accessories/:id", accessoriesConsumablesHandler.DeleteJobAccessoryAPI)

				apiJobAccessories.GET("/:id/consumables", accessoriesConsumablesHandler.GetJobConsumablesAPI)
				apiJobAccessories.POST("/:id/consumables", accessoriesConsumablesHandler.CreateJobConsumableAPI)
				apiJobAccessories.PUT("/consumables/:id", accessoriesConsumablesHandler.UpdateJobConsumableAPI)
				apiJobAccessories.DELETE("/consumables/:id", accessoriesConsumablesHandler.DeleteJobConsumableAPI)
			}

			// Scanning API (for WarehouseCore integration)
			apiScan := api.Group("/scan")
			{
				apiScan.POST("/accessory", accessoriesConsumablesHandler.ScanAccessoryAPI)
				apiScan.POST("/consumable", accessoriesConsumablesHandler.ScanConsumableAPI)
			}

			// Cable API - removed, now handled by WarehouseCore
			// Only keeping read-only endpoints if needed by jobs/invoices
			// apiCables := api.Group("/cables")
			// {
			// 	apiCables.GET("", cableHandler.ListCablesAPI)
			// 	apiCables.POST("", cableHandler.CreateCableAPI)
			// 	apiCables.GET("/:id", cableHandler.GetCableAPI)
			// 	apiCables.PUT("/:id", cableHandler.UpdateCableAPI)
			// 	apiCables.DELETE("/:id", cableHandler.DeleteCableAPI)
			// 	apiCables.GET("/types", cableHandler.GetCableTypesAPI)
			// 	apiCables.GET("/connectors", cableHandler.GetCableConnectorsAPI)
			// }

			// Customer API
			apiCustomers := api.Group("/customers")
			{
				apiCustomers.GET("", customerHandler.ListCustomersAPI)
				apiCustomers.POST("", customerHandler.CreateCustomerAPI)
				apiCustomers.GET("/:id", customerHandler.GetCustomerAPI)
				apiCustomers.PUT("/:id", customerHandler.UpdateCustomerAPI)
				apiCustomers.DELETE("/:id", customerHandler.DeleteCustomerAPI)
			}

			// Case API - Removed: Now handled by WarehouseCore
			// Cases are fully managed in WarehouseCore
			// All case CRUD operations, device mapping, and queries should be directed to WarehouseCore
			// apiCases := api.Group("/cases")
			// {
			// 	apiCases.GET("", caseHandler.ListCasesAPI)
			// 	apiCases.POST("", caseHandler.CreateCaseAPI)
			// 	apiCases.GET("/:id", caseHandler.GetCaseAPI)
			// 	apiCases.PUT("/:id", caseHandler.UpdateCaseAPI)
			// 	apiCases.DELETE("/:id", caseHandler.DeleteCaseAPI)
			// 	apiCases.GET("/:id/devices", caseHandler.GetCaseDevicesAPI)
			// 	apiCases.DELETE("/:id/devices/:deviceId", caseHandler.RemoveDeviceFromCase)
			// 	apiCases.GET("/devices/tree", caseHandler.GetAvailableDevicesWithCaseInfo)
			// }

			// Workflow API
			apiWorkflow := api.Group("/workflow")
			{
				packagesDisabled := deprecatedFeatureHandler("Equipment packages API")
				apiWorkflow.Any("/packages", packagesDisabled)
				apiWorkflow.Any("/packages/*path", packagesDisabled)
			}

			// Rental Equipment API
			apiRentalEquipment := api.Group("/rental-equipment")
			{
				apiRentalEquipment.GET("", rentalEquipmentHandler.GetRentalEquipmentAPI)
				apiRentalEquipment.POST("", rentalEquipmentHandler.CreateRentalEquipment)
				apiRentalEquipment.PUT("/:id", rentalEquipmentHandler.UpdateRentalEquipment)
				apiRentalEquipment.DELETE("/:id", rentalEquipmentHandler.DeleteRentalEquipment)
				apiRentalEquipment.POST("/add-to-job", rentalEquipmentHandler.AddRentalToJob)
				apiRentalEquipment.POST("/manual-entry", rentalEquipmentHandler.CreateManualRentalEntry)
				apiRentalEquipment.GET("/job/:jobId", rentalEquipmentHandler.GetJobRentalEquipment)
				apiRentalEquipment.DELETE("/job/:jobId/equipment/:equipmentId", rentalEquipmentHandler.RemoveRentalFromJob)
				apiRentalEquipment.GET("/analytics", rentalEquipmentHandler.GetRentalAnalyticsAPI)
			}

			// Document API
			apiDocuments := api.Group("/documents")
			{
				apiDocuments.GET("", documentHandler.ListDocumentsAPI)
				apiDocuments.GET("/stats", documentHandler.GetDocumentStats)
				apiDocuments.GET("/:id", documentHandler.GetDocument)
				apiDocuments.DELETE("/:id", documentHandler.DeleteDocument)
			}

			// PDF Processing API
			apiPDF := api.Group("/pdf")
			{
				apiPDF.POST("/upload", pdfHandler.UploadPDF)
				apiPDF.GET("/extraction/:upload_id", pdfHandler.GetExtractionResult)
				apiPDF.POST("/mapping", pdfHandler.SaveProductMapping)
				apiPDF.GET("/suggestions", pdfHandler.GetProductSuggestions)
				apiPDF.PUT("/items/:item_id/mapping", pdfHandler.UpdateItemMapping)
				apiPDF.POST("/auto-map/:extraction_id", pdfHandler.RunAutoMapping)
				apiPDF.POST("/manual-map/:item_id", pdfHandler.SaveManualMapping)
				apiPDF.GET("/products/search", pdfHandler.SearchProducts)
				apiPDF.GET("/packages/search", pdfHandler.SearchPackages)
				apiPDF.GET("/rental-equipment/search", pdfHandler.SearchRentalEquipment)
				apiPDF.GET("/service-items", pdfHandler.ListServiceItems)
				apiPDF.GET("/service-items/:id", pdfHandler.GetServiceItem)
				apiPDF.GET("/service-items/search", pdfHandler.SearchServiceItems)
				apiPDF.POST("/service-items", pdfHandler.CreateServiceItemQuick)
				apiPDF.PUT("/service-items/:id", pdfHandler.UpdateServiceItem)
				apiPDF.DELETE("/service-items/:id", pdfHandler.DeleteServiceItem)
				apiPDF.GET("/customers/search", pdfHandler.SearchCustomers)
				apiPDF.GET("/extractions/:extraction_id/duplicates", pdfHandler.GetDuplicateJobCandidates)
				apiPDF.POST("/customer-map/:extraction_id", pdfHandler.SaveCustomerMapping)
				apiPDF.POST("/customers/from-extraction/:extraction_id", pdfHandler.CreateCustomerFromExtraction)
				apiPDF.POST("/extractions/:extraction_id/finalize", pdfHandler.FinalizeExtraction)
				// File Pool integration routes
				apiPDF.POST("/from-pool/:documentID", pdfHandler.ProcessPoolDocument)
				apiPDF.GET("/pool-documents", pdfHandler.GetPoolDocumentsForOCR)
				apiPDF.GET("/mappings", pdfHandler.GetAllMappingsAPI)
				apiPDF.DELETE("/mappings/:id", pdfHandler.DeleteMappingAPI)
				apiPDF.PUT("/mappings/:id", pdfHandler.UpdateMappingAPI)
				apiPDF.GET("/extractions/:extraction_id/preview", pdfHandler.GetExtractionPreview)
				apiPDF.POST("/mappings-create", pdfHandler.CreateMappingAPI)
				apiPDF.PUT("/package-mappings/:id", pdfHandler.UpdatePackageMappingAPI)
				apiPDF.DELETE("/package-mappings/:id", pdfHandler.DeletePackageMappingAPI)
				apiPDF.PUT("/customer-mappings/:id", pdfHandler.UpdateCustomerMappingAPI)
				apiPDF.DELETE("/customer-mappings/:id", pdfHandler.DeleteCustomerMappingAPI)
				apiPDF.POST("/product-quick-create", pdfHandler.CreateProductQuick)
				apiPDF.POST("/package-quick-create", pdfHandler.CreatePackageQuick)
				apiPDF.POST("/rental-equipment-quick-create", pdfHandler.CreateRentalEquipmentQuick)
			}

			// Financial API
			apiFinancial := api.Group("/financial")
			{
				apiFinancial.GET("/transactions", financialHandler.ListTransactionsAPI)
				apiFinancial.GET("/stats", financialHandler.GetFinancialStatsAPI)
				apiFinancial.GET("/revenue-report", financialHandler.GetRevenueReport)
				apiFinancial.GET("/payment-report", financialHandler.GetPaymentReport)
			}

			// Invoice API (using NEW fixed invoice system)
			apiInvoices := api.Group("/invoices")
			{
				invoicesDisabled := deprecatedFeatureHandler("Invoices API")
				apiInvoices.Any("", invoicesDisabled)
				apiInvoices.Any("/*path", invoicesDisabled)
			}

			// User preferences API
			apiUsers := api.Group("/users")
			{
				apiMe := apiUsers.Group("/me")
				{
					dashboard := apiMe.Group("/dashboard")
					{
						dashboard.GET("/widgets", homeHandler.GetDashboardWidgetPreferences)
						dashboard.PUT("/widgets", homeHandler.UpdateDashboardWidgetPreferences)
					}
				}
			}

			// Security API
			apiSecurity := api.Group("/security")
			{
				// Audit API
				auditAPI := apiSecurity.Group("/audit")
				{
					auditAPI.GET("", securityHandler.GetAuditLogs)
					auditAPI.GET("/:id", securityHandler.GetAuditLog)
					auditAPI.GET("/export", securityHandler.ExportAuditLogs)
				}

				// Auth API
				authAPI := apiSecurity.Group("/auth")
				{
					authAPI.GET("/users", authHandler.ListUsersAPI)
					authAPI.POST("/users", authHandler.CreateUserAPI)
					authAPI.GET("/users/:id", authHandler.GetUserAPI)
					authAPI.PUT("/users/:id", authHandler.UpdateUserAPI)
					authAPI.DELETE("/users/:id", authHandler.DeleteUserAPI)
				}
			}

			// Skills API
			apiSkills := api.Group("/skills")
			{
				apiSkills.GET("", skillHandler.List)
				apiSkills.POST("", skillHandler.Create)
				apiSkills.PUT("/:id", skillHandler.Update)
				apiSkills.DELETE("/:id", skillHandler.Delete)
			}

			// Employees API
			apiEmployees := api.Group("/employees")
			{
				apiEmployees.GET("", employeeHandler.List)
				apiEmployees.GET("/active", employeeHandler.ListActive)
				apiEmployees.GET("/:id", employeeHandler.GetByID)
				apiEmployees.POST("", employeeHandler.Create)
				apiEmployees.PUT("/:id", employeeHandler.Update)
				apiEmployees.DELETE("/:id", employeeHandler.Delete)
			}

			// Job-Employee assignment API
			api.GET("/jobs/:id/employees", jobHandler.ListJobEmployees)
			api.POST("/jobs/:id/employees", jobHandler.AssignEmployee)
			api.DELETE("/jobs/:id/employees/:employeeId", jobHandler.RemoveEmployee)

			// Venues API
			apiVenues := api.Group("/venues")
			{
				apiVenues.GET("", venueHandler.List)
				apiVenues.POST("", venueHandler.Create)
				apiVenues.PUT("/:id", venueHandler.Update)
				apiVenues.DELETE("/:id", venueHandler.Delete)
			}

			// Profile security routes (also exposed under /api/v1 for Cores proxy)
			apiProfile := api.Group("/profile")
			{
				apiPasskeys := apiProfile.Group("/passkeys")
				{
					apiPasskeys.POST("/start-registration", profileHandler.StartPasskeyRegistration)
					apiPasskeys.POST("/complete-registration", profileHandler.CompletePasskeyRegistration)
					apiPasskeys.GET("", profileHandler.ListUserPasskeys)
					apiPasskeys.DELETE("/:id", profileHandler.DeletePasskey)
				}
				apiTwoFA := apiProfile.Group("/2fa")
				{
					apiTwoFA.GET("/status", profileHandler.Get2FAStatus)
					apiTwoFA.POST("/setup", profileHandler.Setup2FA)
					apiTwoFA.POST("/verify", profileHandler.Verify2FA)
					apiTwoFA.POST("/disable", profileHandler.Disable2FA)
				}
			}

			// M365 Settings
			apiSettingsM365 := api.Group("/settings")
			{
				apiSettingsM365.GET("/m365", m365SettingsHandler.GetM365Settings)
				apiSettingsM365.PUT("/m365", m365SettingsHandler.UpdateM365Settings)
			}
		}

		// Additional API routes (outside v1 group for legacy compatibility)
		legacyAPI := protected.Group("/api")
		{
			// Legacy Invoice API (using NEW fixed invoice system)
			legacyInvoicesDisabled := deprecatedFeatureHandler("Invoices API")
			legacyAPI.Any("/invoices", legacyInvoicesDisabled)
			legacyAPI.Any("/invoices/*path", legacyInvoicesDisabled)

			// Legacy Rental Equipment API
			legacyAPI.GET("/rental-equipment", rentalEquipmentHandler.GetRentalEquipmentAPI)
			legacyAPI.POST("/rental-equipment", rentalEquipmentHandler.CreateRentalEquipment)
			legacyAPI.PUT("/rental-equipment/:id", rentalEquipmentHandler.UpdateRentalEquipment)
			legacyAPI.DELETE("/rental-equipment/:id", rentalEquipmentHandler.DeleteRentalEquipment)
			legacyAPI.GET("/rental-equipment/analytics", rentalEquipmentHandler.GetRentalAnalyticsAPI)

			// Job Attachments API
			jobAttachmentsAPI := legacyAPI.Group("/jobs")
			{
				jobAttachmentsAPI.GET("/:jobid/attachments", jobAttachmentHandler.GetJobAttachments)
				jobAttachmentsAPI.POST("/attachments/upload", jobAttachmentHandler.UploadAttachment)
				jobAttachmentsAPI.GET("/attachments/:id/view", jobAttachmentHandler.ViewAttachment)
				jobAttachmentsAPI.GET("/attachments/:id/download", jobAttachmentHandler.DownloadAttachment)
				jobAttachmentsAPI.DELETE("/attachments/:id", jobAttachmentHandler.DeleteAttachment)
				jobAttachmentsAPI.PUT("/attachments/:id/description", jobAttachmentHandler.UpdateAttachmentDescription)
			}

			// PDF Processing API
			pdfAPI := legacyAPI.Group("/pdf")
			{
				pdfAPI.POST("/upload", pdfHandler.UploadPDF)
				pdfAPI.GET("/extraction/:upload_id", pdfHandler.GetExtractionResult)
				pdfAPI.POST("/mapping", pdfHandler.SaveProductMapping)
				pdfAPI.GET("/suggestions", pdfHandler.GetProductSuggestions)
				pdfAPI.PUT("/items/:item_id/mapping", pdfHandler.UpdateItemMapping)
				pdfAPI.POST("/auto-map/:extraction_id", pdfHandler.RunAutoMapping)
				pdfAPI.POST("/manual-map/:item_id", pdfHandler.SaveManualMapping)
				pdfAPI.GET("/products/search", pdfHandler.SearchProducts)
				pdfAPI.GET("/packages/search", pdfHandler.SearchPackages)
				pdfAPI.GET("/rental-equipment/search", pdfHandler.SearchRentalEquipment)
				pdfAPI.GET("/service-items", pdfHandler.ListServiceItems)
				pdfAPI.GET("/service-items/:id", pdfHandler.GetServiceItem)
				pdfAPI.GET("/service-items/search", pdfHandler.SearchServiceItems)
				pdfAPI.POST("/service-items", pdfHandler.CreateServiceItemQuick)
				pdfAPI.GET("/customers/search", pdfHandler.SearchCustomers)
				pdfAPI.GET("/extractions/:extraction_id/duplicates", pdfHandler.GetDuplicateJobCandidates)
				pdfAPI.POST("/customer-map/:extraction_id", pdfHandler.SaveCustomerMapping)
				pdfAPI.POST("/customers/from-extraction/:extraction_id", pdfHandler.CreateCustomerFromExtraction)
				pdfAPI.POST("/extractions/:extraction_id/finalize", pdfHandler.FinalizeExtraction)
				// File Pool integration routes
				pdfAPI.POST("/from-pool/:documentID", pdfHandler.ProcessPoolDocument)
				pdfAPI.GET("/pool-documents", pdfHandler.GetPoolDocumentsForOCR)
				pdfAPI.GET("/mappings", pdfHandler.GetAllMappingsAPI)
				pdfAPI.DELETE("/mappings/:id", pdfHandler.DeleteMappingAPI)
				pdfAPI.PUT("/mappings/:id", pdfHandler.UpdateMappingAPI)
				pdfAPI.GET("/extractions/:extraction_id/preview", pdfHandler.GetExtractionPreview)
				pdfAPI.POST("/mappings-create", pdfHandler.CreateMappingAPI)
				pdfAPI.PUT("/package-mappings/:id", pdfHandler.UpdatePackageMappingAPI)
				pdfAPI.DELETE("/package-mappings/:id", pdfHandler.DeletePackageMappingAPI)
				pdfAPI.PUT("/customer-mappings/:id", pdfHandler.UpdateCustomerMappingAPI)
				pdfAPI.DELETE("/customer-mappings/:id", pdfHandler.DeleteCustomerMappingAPI)
				pdfAPI.POST("/product-quick-create", pdfHandler.CreateProductQuick)
				pdfAPI.POST("/package-quick-create", pdfHandler.CreatePackageQuick)
				pdfAPI.POST("/rental-equipment-quick-create", pdfHandler.CreateRentalEquipmentQuick)
			}

			// Company settings API - NOW ACTIVE
			legacyAPI.GET("/company-settings", companyHandler.GetCompanySettings)
			legacyAPI.PUT("/company-settings", companyHandler.UpdateCompanySettings)
			legacyAPI.POST("/company-settings/logo", companyHandler.UploadCompanyLogo)
			legacyAPI.DELETE("/company-settings/logo", companyHandler.DeleteCompanyLogo)

			// SMTP Configuration API
			settingsAPI := legacyAPI.Group("/settings")
			{
				settingsAPI.GET("/smtp", companyHandler.GetSMTPConfig)
				settingsAPI.POST("/smtp", companyHandler.UpdateSMTPConfig)
				settingsAPI.POST("/smtp/test", companyHandler.TestSMTPConnection)
			}

			// TODO: Invoice settings API - temporarily disabled
			// Will be re-implemented in new system when needed
			// legacyAPI.GET("/invoice-settings", invoiceHandler.InvoiceSettingsForm)
			// legacyAPI.PUT("/invoice-settings", invoiceHandler.UpdateInvoiceSettings)

			// TODO: Email API - temporarily disabled
			// Will be re-implemented in new system when needed
			// legacyAPI.POST("/test-email", invoiceHandler.TestEmailSettings)

			// Security API - Legacy routes kept for backward compatibility with frontend
			apiSecurity := legacyAPI.Group("/security")
			{
				// Roles API
				apiSecurity.GET("/roles", securityHandler.GetRoles)
				apiSecurity.GET("/roles/:id", securityHandler.GetRole)
				apiSecurity.POST("/roles", securityHandler.CreateRole)
				apiSecurity.PUT("/roles/:id", securityHandler.UpdateRole)
				apiSecurity.DELETE("/roles/:id", securityHandler.DeleteRole)

				// User roles API
				apiSecurity.GET("/users/:userId/roles", securityHandler.GetUserRoles)
				apiSecurity.POST("/users/:userId/roles", securityHandler.AssignUserRole)
				apiSecurity.DELETE("/users/:userId/roles/:roleId", securityHandler.RevokeUserRole)

				// Audit API
				apiSecurity.GET("/audit", securityHandler.GetAuditLogs)
				apiSecurity.GET("/audit/:id", securityHandler.GetAuditLog)
				apiSecurity.GET("/audit/export", securityHandler.ExportAuditLogs)

				// Permissions API
				apiSecurity.GET("/permissions", securityHandler.GetPermissions)
				apiSecurity.GET("/permissions/definitions", securityHandler.GetPermissionDefinitionsAPI)
				apiSecurity.GET("/permissions/check", securityHandler.CheckPermission)
			}
		}
	}
}

// createDefaultTemplate creates a default invoice template if none exists
func createDefaultTemplate(templateHandler *handlers.InvoiceTemplateHandler, repo *repository.InvoiceRepositoryNew) error {
	// Check if any templates exist
	templates, err := repo.GetAllTemplates()
	if err != nil {
		return fmt.Errorf("failed to check existing templates: %v", err)
	}

	// If templates exist, skip creation
	if len(templates) > 0 {
		logger.LogInfo("Templates already exist (%d), skipping default template creation", len(templates))
		return nil
	}

	// Create a default German standard template
	description := "Standard German invoice template compliant with DIN 5008"
	cssStyles := `{"templateType":"german-din","primaryFont":"Arial","headerFontSize":"18","bodyFontSize":"12","primaryColor":"#2563eb","textColor":"#000000","backgroundColor":"#ffffff","pageMargins":"20","elementSpacing":"15","borderStyle":"solid"}`

	defaultTemplate := &models.InvoiceTemplate{
		Name:         "German Standard (DIN 5008)",
		Description:  &description,
		HTMLTemplate: getDefaultTemplateHTML(),
		CSSStyles:    &cssStyles,
		IsDefault:    true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = repo.CreateTemplate(defaultTemplate)
	if err != nil {
		return fmt.Errorf("failed to create default template: %v", err)
	}

	logger.LogInfo("Successfully created default invoice template")
	return nil
}

// getDefaultTemplateHTML returns the HTML for the default German standard template
func getDefaultTemplateHTML() string {
	return `<div style="padding: 20mm; font-family: Arial, sans-serif; font-size: 12px;">
    <!-- Company Header -->
    <div style="text-align: right; margin-bottom: 30px;">
        <div style="font-weight: bold; font-size: 16px;">{{.company.CompanyName}}</div>
        <div>{{if .company.AddressLine1}}{{.company.AddressLine1}}{{end}}</div>
        <div>{{if .company.PostalCode}}{{.company.PostalCode}} {{end}}{{if .company.City}}{{.company.City}}{{end}}</div>
        <div>{{if .company.Phone}}Tel: {{.company.Phone}}{{end}}</div>
        <div>{{if .company.Email}}E-Mail: {{.company.Email}}{{end}}</div>
    </div>

    <!-- Sender Address Line -->
    <div style="font-size: 8px; margin-bottom: 10px; border-bottom: 1px solid #000; padding-bottom: 5px;">
        {{.company.CompanyName}}, {{if .company.AddressLine1}}{{.company.AddressLine1}}, {{end}}{{if .company.PostalCode}}{{.company.PostalCode}} {{end}}{{if .company.City}}{{.company.City}}{{end}}
    </div>

    <!-- Customer Address -->
    <div style="margin-bottom: 30px;">
        <div style="font-weight: bold;">{{.customer.GetDisplayName}}</div>
        <div>{{if .customer.Street}}{{.customer.Street}}{{if .customer.HouseNumber}} {{.customer.HouseNumber}}{{end}}{{end}}</div>
        <div>{{if .customer.ZIP}}{{.customer.ZIP}} {{end}}{{if .customer.City}}{{.customer.City}}{{end}}</div>
    </div>

    <!-- Invoice Header -->
    <h1 style="font-size: 24px; font-weight: bold; margin-bottom: 20px;">RECHNUNG</h1>

    <!-- Invoice Details -->
    <table style="width: 100%; margin-bottom: 20px; border-collapse: collapse;">
        <tr>
            <td style="width: 30%; padding: 5px 0;">Rechnungsnummer:</td>
            <td style="font-weight: bold;">{{.invoice.InvoiceNumber}}</td>
        </tr>
        <tr>
            <td style="padding: 5px 0;">Rechnungsdatum:</td>
            <td>{{.invoice.IssueDate.Format "02.01.2006"}}</td>
        </tr>
        <tr>
            <td style="padding: 5px 0;">Fälligkeitsdatum:</td>
            <td>{{.invoice.DueDate.Format "02.01.2006"}}</td>
        </tr>
        <tr>
            <td style="padding: 5px 0;">Kundennummer:</td>
            <td>{{.customer.CustomerID}}</td>
        </tr>
    </table>

    <!-- Line Items -->
    <table style="width: 100%; border-collapse: collapse; margin-bottom: 20px;">
        <thead>
            <tr style="background: #f5f5f5;">
                <th style="border: 1px solid #000; padding: 8px; text-align: left;">Pos.</th>
                <th style="border: 1px solid #000; padding: 8px; text-align: left;">Beschreibung</th>
                <th style="border: 1px solid #000; padding: 8px; text-align: center;">Menge</th>
                <th style="border: 1px solid #000; padding: 8px; text-align: right;">Einzelpreis</th>
                <th style="border: 1px solid #000; padding: 8px; text-align: right;">Gesamtpreis</th>
            </tr>
        </thead>
        <tbody>
            {{range $index, $item := .invoice.LineItems}}
            <tr>
                <td style="border: 1px solid #000; padding: 8px;">{{add $index 1}}</td>
                <td style="border: 1px solid #000; padding: 8px;">{{$item.Description}}</td>
                <td style="border: 1px solid #000; padding: 8px; text-align: center;">{{$item.Quantity}}</td>
                <td style="border: 1px solid #000; padding: 8px; text-align: right;">{{printf "%.2f" $item.UnitPrice}} €</td>
                <td style="border: 1px solid #000; padding: 8px; text-align: right;">{{printf "%.2f" $item.TotalPrice}} €</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <!-- Totals -->
    <div style="text-align: right; margin-bottom: 30px;">
        <table style="width: 200px; margin-left: auto; border-collapse: collapse;">
            <tr>
                <td style="padding: 5px 10px; border-bottom: 1px solid #ddd;">Nettobetrag:</td>
                <td style="text-align: right; padding: 5px 10px; border-bottom: 1px solid #ddd;">{{printf "%.2f" .invoice.Subtotal}} €</td>
            </tr>
            <tr>
                <td style="padding: 5px 10px; border-bottom: 1px solid #ddd;">MwSt. ({{.invoice.TaxRate}}%):</td>
                <td style="text-align: right; padding: 5px 10px; border-bottom: 1px solid #ddd;">{{printf "%.2f" .invoice.TaxAmount}} €</td>
            </tr>
            <tr style="font-weight: bold; border-top: 2px solid #000;">
                <td style="padding: 8px 10px;">Gesamtbetrag:</td>
                <td style="text-align: right; padding: 8px 10px;">{{printf "%.2f" .invoice.TotalAmount}} €</td>
            </tr>
        </table>
    </div>

    <!-- Footer -->
    <div style="font-size: 10px; margin-top: 40px; border-top: 1px solid #ddd; padding-top: 20px;">
        <div style="text-align: center;">
            <div>{{if .company.TaxNumber}}Steuernummer: {{.company.TaxNumber}}{{end}}{{if and .company.TaxNumber .company.VATNumber}} | {{end}}{{if .company.VATNumber}}USt-IdNr.: {{.company.VATNumber}}{{end}}</div>
            <div style="margin-top: 10px;">
                Zahlungsziel: 14 Tage netto | Vielen Dank für Ihr Vertrauen!
            </div>
        </div>
    </div>
</div>`
}
