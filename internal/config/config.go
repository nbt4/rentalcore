package config

import (
	"fmt"
	"encoding/json"
	"os"
	"strconv"
	"time"
	
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Database DatabaseConfig `json:"database"`
	Server   ServerConfig   `json:"server"`
	UI       UIConfig       `json:"ui"`
	Email    EmailConfig    `json:"email"`
	Invoice  InvoiceConfig  `json:"invoice"`
	PDF      PDFConfig      `json:"pdf"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
	Backup   BackupConfig   `json:"backup"`
	Features FeaturesConfig `json:"features"`
	M365     M365Config     `json:"m365"`
}

type DatabaseConfig struct {
// PostgreSQL Configuration
Host     string `json:"host"`
Port     int    `json:"port"`
Name     string `json:"name"`
User     string `json:"user"`
Password string `json:"password"`
SSLMode  string `json:"sslmode"`

// Connection Pool
MaxOpenConns    int           `json:"max_open_conns"`
MaxIdleConns    int           `json:"max_idle_conns"`
ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`

// Query settings
SlowQueryThreshold time.Duration   `json:"slow_query_threshold"`
EnableQueryLogging bool            `json:"enable_query_logging"`
LogLevel           logger.LogLevel `json:"-"`
PrepareStmt        bool            `json:"prepare_stmt"`
DisableForeignKeyConstraintWhenMigrating bool `json:"disable_fk_when_migrating"`
}
// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
        return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
                c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}




type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

type UIConfig struct {
	ThemeDark      string            `json:"theme_dark"`
	ThemeLight     string            `json:"theme_light"`
	CurrentTheme   string            `json:"current_theme"`
	Colors         map[string]string `json:"colors"`
	AutoSave       bool              `json:"auto_save"`
	AutoSaveInterval int             `json:"auto_save_interval"`
	CacheTimeout   int               `json:"cache_timeout"`
	WindowWidth    int               `json:"window_width"`
	WindowHeight   int               `json:"window_height"`
}

type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
}

type InvoiceConfig struct {
	DefaultTaxRate          float64 `json:"default_tax_rate"`
	DefaultPaymentTerms     int     `json:"default_payment_terms"`
	AutoCalculateRentalDays bool    `json:"auto_calculate_rental_days"`
	ShowLogoOnInvoice       bool    `json:"show_logo_on_invoice"`
	InvoiceNumberPrefix     string  `json:"invoice_number_prefix"`
	InvoiceNumberFormat     string  `json:"invoice_number_format"`
	CurrencySymbol          string  `json:"currency_symbol"`
	CurrencyCode            string  `json:"currency_code"`
	DateFormat              string  `json:"date_format"`
}

type PDFConfig struct {
	Generator string            `json:"generator"`
	PaperSize string            `json:"paper_size"`
	Margins   map[string]string `json:"margins"`
}

type SecurityConfig struct {
	SessionTimeout    int    `json:"session_timeout"`
	PasswordMinLength int    `json:"password_min_length"`
	MaxLoginAttempts  int    `json:"max_login_attempts"`
	LockoutDuration   int    `json:"lockout_duration"`
	EncryptionKey     string `json:"encryption_key"`
}

type LoggingConfig struct {
	Level      string `json:"level"`
	File       string `json:"file"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
}

type BackupConfig struct {
	Enabled       bool   `json:"enabled"`
	Interval      int    `json:"interval"`
	RetentionDays int    `json:"retention_days"`
	Path          string `json:"path"`
}

type FeaturesConfig struct {
	// ScannerEnabled field deprecated - scanner functionality removed
	// Kept for backwards compatibility with existing config files
	ScannerEnabled bool `json:"scanner_enabled"`
}

func LoadConfig(path string) (*Config, error) {
	// Start with default config
	config := getDefaultConfig()
	
	// Override with environment variables if they exist
	loadFromEnvironment(config)
	
	// Try to load from file if it exists
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return nil, err
		}
		// Override again with environment variables to give them priority
		loadFromEnvironment(config)
	}

	// FIXED: Validate critical security settings are not empty
	if config.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable is required")
	}
	if config.Security.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY environment variable is required")
	}

	return config, nil
}

func (c *Config) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

func getDefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
Host:               "localhost",
Port:               5432,
Name:               "rentalcore",
User:               "rentalcore",
Password:           "", // FIXED: removed hardcoded password, must be set via DB_PASSWORD env
SSLMode:            "disable",
MaxOpenConns:       25,
MaxIdleConns:       10,
ConnMaxLifetime:    time.Hour,
ConnMaxIdleTime:    30 * time.Minute,
SlowQueryThreshold: 500 * time.Millisecond,
EnableQueryLogging: false,
LogLevel:           logger.Warn,
PrepareStmt:        true,
DisableForeignKeyConstraintWhenMigrating: true,
},
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		UI: UIConfig{
			ThemeDark:        "darkly",
			ThemeLight:       "flatly",
			CurrentTheme:     "dark",
			AutoSave:         true,
			AutoSaveInterval: 300,
			CacheTimeout:     300,
			WindowWidth:      1400,
			WindowHeight:     800,
			Colors: map[string]string{
				"primary":     "#007bff",
				"background":  "#ffffff",
				"text":        "#000000",
				"selection":   "#e9ecef",
				"success":     "#28a745",
				"error":       "#dc3545",
				"warning":     "#ffc107",
				"dark_bg":     "#2b2b2b",
				"dark_text":   "#ffffff",
			},
		},
		Email: EmailConfig{
			SMTPHost:     "localhost",
			SMTPPort:     587,
			SMTPUsername: "",
			SMTPPassword: "",
			FromEmail:    "noreply@rentalcore.com",
			FromName:     "RentalCore",
			UseTLS:       true,
		},
		Invoice: InvoiceConfig{
			DefaultTaxRate:          19.0,
			DefaultPaymentTerms:     30,
			AutoCalculateRentalDays: true,
			ShowLogoOnInvoice:       true,
			InvoiceNumberPrefix:     "INV-",
			InvoiceNumberFormat:     "{prefix}{year}{month}{sequence:4}",
			CurrencySymbol:          "€",
			CurrencyCode:            "EUR",
			DateFormat:              "DD.MM.YYYY",
		},
		PDF: PDFConfig{
			Generator: "auto",
			PaperSize: "A4",
			Margins: map[string]string{
				"top":    "1cm",
				"bottom": "1cm",
				"left":   "1cm",
				"right":  "1cm",
			},
		},
		Security: SecurityConfig{
			SessionTimeout:    3600,
			PasswordMinLength: 8,
			MaxLoginAttempts:  5,
			LockoutDuration:   900,
			EncryptionKey:     "", // FIXED: removed hardcoded demo key, must be set via ENCRYPTION_KEY env
		},
		Logging: LoggingConfig{
			Level:      "info",
			File:       "logs/app.log",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
		},
		Backup: BackupConfig{
			Enabled:       true,
			Interval:      86400,
			RetentionDays: 30,
			Path:          "backups/",
		},
		Features: FeaturesConfig{
			ScannerEnabled: false, // Scanner functionality removed
		},
	}
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *Config) {
	// PostgreSQL Database configuration
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		config.Database.Name = name
	}
	if user := os.Getenv("DB_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	if sslMode := os.Getenv("DB_SSLMODE"); sslMode != "" {
		config.Database.SSLMode = sslMode
	}
	if maxOpenConns := os.Getenv("DB_MAX_OPEN_CONNS"); maxOpenConns != "" {
		if moc, err := strconv.Atoi(maxOpenConns); err == nil {
			config.Database.MaxOpenConns = moc
		}
	}

	// Server configuration
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	// Security configuration
	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		config.Security.EncryptionKey = key
	}
	if timeout := os.Getenv("SESSION_TIMEOUT"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			config.Security.SessionTimeout = t
		}
	}

	// Email configuration
	if host := os.Getenv("SMTP_HOST"); host != "" {
		config.Email.SMTPHost = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Email.SMTPPort = p
		}
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		config.Email.SMTPUsername = username
	}
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		config.Email.SMTPPassword = password
	}
	if fromEmail := os.Getenv("FROM_EMAIL"); fromEmail != "" {
		config.Email.FromEmail = fromEmail
	}
	if fromName := os.Getenv("FROM_NAME"); fromName != "" {
		config.Email.FromName = fromName
	}
	if useTLS := os.Getenv("USE_TLS"); useTLS != "" {
		config.Email.UseTLS = useTLS == "true"
	}

	// Invoice configuration
	if taxRate := os.Getenv("DEFAULT_TAX_RATE"); taxRate != "" {
		if rate, err := strconv.ParseFloat(taxRate, 64); err == nil {
			config.Invoice.DefaultTaxRate = rate
		}
	}
	if paymentTerms := os.Getenv("DEFAULT_PAYMENT_TERMS"); paymentTerms != "" {
		if terms, err := strconv.Atoi(paymentTerms); err == nil {
			config.Invoice.DefaultPaymentTerms = terms
		}
	}
	if symbol := os.Getenv("CURRENCY_SYMBOL"); symbol != "" {
		config.Invoice.CurrencySymbol = symbol
	}
	if code := os.Getenv("CURRENCY_CODE"); code != "" {
		config.Invoice.CurrencyCode = code
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if file := os.Getenv("LOG_FILE"); file != "" {
		config.Logging.File = file
	}

	// Backup configuration
	if enabled := os.Getenv("BACKUP_ENABLED"); enabled != "" {
		config.Backup.Enabled = enabled == "true"
	}
	if interval := os.Getenv("BACKUP_INTERVAL"); interval != "" {
		if i, err := strconv.Atoi(interval); err == nil {
			config.Backup.Interval = i
		}
	}
	if retention := os.Getenv("BACKUP_RETENTION_DAYS"); retention != "" {
		if r, err := strconv.Atoi(retention); err == nil {
			config.Backup.RetentionDays = r
		}
	}

	// Features configuration (deprecated)
	// Scanner functionality has been removed, but we keep the config field for backwards compatibility
	if scannerEnabled := os.Getenv("SCANNER_ENABLED"); scannerEnabled != "" {
		config.Features.ScannerEnabled = false // Always false, scanner removed
	}

	config.M365.LoadFromEnv()
}

type M365Config struct {
	TenantID        string `json:"tenant_id"`
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	MailboxID       string `json:"mailbox_id"`
	SyncInterval    string `json:"sync_interval"`
	CalendarMailbox string `json:"calendar_mailbox"`
	AppBaseURL      string `json:"app_base_url"`
}

func (c *M365Config) LoadFromEnv() {
	if v := os.Getenv("M365_TENANT_ID"); v != "" {
		c.TenantID = v
	}
	if v := os.Getenv("M365_CLIENT_ID"); v != "" {
		c.ClientID = v
	}
	if v := os.Getenv("M365_CLIENT_SECRET"); v != "" {
		c.ClientSecret = v
	}
	if v := os.Getenv("M365_SHARED_MAILBOX_ID"); v != "" {
		c.MailboxID = v
	}
	if v := os.Getenv("M365_SYNC_INTERVAL"); v != "" {
		c.SyncInterval = v
	} else {
		c.SyncInterval = "5m"
	}
	if v := os.Getenv("M365_CALENDAR_MAILBOX"); v != "" {
		c.CalendarMailbox = v
	} else {
		c.CalendarMailbox = "events@tsunami-events.de"
	}
	if v := os.Getenv("APP_BASE_URL"); v != "" {
		c.AppBaseURL = v
	}
}

func (c *M365Config) IsConfigured() bool {
	return c.TenantID != "" && c.ClientID != "" && c.ClientSecret != "" && c.MailboxID != ""
}
// GetDatabaseStats returns database connection statistics
func GetDatabaseStats(db *gorm.DB) (map[string]interface{}, error) {
sqlDB, err := db.DB()
if err != nil {
return nil, err
}
stats := sqlDB.Stats()
return map[string]interface{}{
"database_type":        "PostgreSQL",
"max_open_connections": stats.MaxOpenConnections,
"open_connections":     stats.OpenConnections,
"in_use":               stats.InUse,
"idle":                 stats.Idle,
}, nil
}

// ApplyPerformanceIndexes applies database indexes for performance
func ApplyPerformanceIndexes(db *gorm.DB) error {
// PostgreSQL indexes are handled by migrations
return nil
}
