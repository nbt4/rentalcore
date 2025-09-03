package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Note: DatabaseConfig is now defined in config.go to avoid duplication

// GetDatabaseConfig returns database configuration from environment
func GetDatabaseConfig() *DatabaseConfig {
	config := &DatabaseConfig{
		Host:                  getEnv("DB_HOST", "mysql"),
		Port:                  getEnvAsInt("DB_PORT", 3306),
		Database:              getEnv("DB_DATABASE", "rentalcore"),
		Username:              getEnv("DB_USERNAME", "rentalcore_user"),
		Password:              getEnv("DB_PASSWORD", "web"),
		MaxOpenConns:          getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:          getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime:       getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime:       getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		SlowQueryThreshold:    getEnvAsDuration("DB_SLOW_QUERY_THRESHOLD", 500*time.Millisecond),
		EnableQueryLogging:    getEnvAsBool("DB_ENABLE_QUERY_LOGGING", false),
		PrepareStmt:           getEnvAsBool("DB_PREPARE_STMT", true),
		DisableForeignKeyConstraintWhenMigrating: getEnvAsBool("DB_DISABLE_FK_WHEN_MIGRATING", true),
	}

	// Set log level based on environment
	if getEnvAsBool("DB_DEBUG", false) {
		config.LogLevel = logger.Info
	} else {
		config.LogLevel = logger.Warn
	}

	return config
}

// ConnectDatabase connects to the database with optimized settings
func ConnectDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	// Build DSN with performance optimizations
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s&interpolateParams=true",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	// Configure GORM with performance settings
	gormConfig := &gorm.Config{
		PrepareStmt:                              config.PrepareStmt,
		DisableForeignKeyConstraintWhenMigrating: config.DisableForeignKeyConstraintWhenMigrating,
		Logger:                                   createLogger(config),
	}

	// Connect to database
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool parameters for optimal performance
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connected successfully with %d max connections", config.MaxOpenConns)
	
	return db, nil
}

// createLogger creates a configured logger for GORM
func createLogger(config *DatabaseConfig) logger.Interface {
	logConfig := logger.Config{
		SlowThreshold:             config.SlowQueryThreshold,
		LogLevel:                  config.LogLevel,
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	}

	if config.EnableQueryLogging {
		return logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logConfig,
		)
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             config.SlowQueryThreshold,
			LogLevel:                  logger.Error, // Only log errors in production
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

// ApplyPerformanceIndexes applies database indexes for performance
func ApplyPerformanceIndexes(db *gorm.DB) error {
	log.Println("Applying performance indexes...")

	// Helper function to create index with error handling
	createIndex := func(indexName, tableName, columns string) {
		// First check if index exists
		var exists bool
		checkSQL := fmt.Sprintf("SHOW INDEX FROM %s WHERE Key_name = '%s'", tableName, indexName)
		
		rows, err := db.Raw(checkSQL).Rows()
		if err == nil {
			if rows.Next() {
				exists = true
			}
			rows.Close()
		}
		
		if !exists {
			indexSQL := fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, tableName, columns)
			if err := db.Exec(indexSQL).Error; err != nil {
				log.Printf("Warning: Failed to create index %s: %v", indexName, err)
			} else {
				log.Printf("Created index: %s", indexName)
			}
		}
	}

	// Apply indexes
	createIndex("idx_devices_productid", "devices", "productID")
	createIndex("idx_devices_status", "devices", "status")
	createIndex("idx_devices_search", "devices", "deviceID, serialnumber")
	createIndex("idx_jobdevices_deviceid", "jobdevices", "deviceID")
	createIndex("idx_jobdevices_jobid", "jobdevices", "jobID")
	createIndex("idx_jobdevices_composite", "jobdevices", "deviceID, jobID")
	createIndex("idx_jobs_customerid", "jobs", "customerID")
	createIndex("idx_jobs_statusid", "jobs", "statusID")
	createIndex("idx_customers_search_company", "customers", "companyname")
	createIndex("idx_customers_search_name", "customers", "firstname, lastname")
	createIndex("idx_customers_email", "customers", "email")
	createIndex("idx_products_categoryid", "products", "categoryID")
	createIndex("idx_products_status", "products", "status")
	createIndex("idx_devices_product_status", "devices", "productID, status")

	log.Println("Performance indexes applied successfully")
	return nil
}

// GetDatabaseStats returns database connection statistics
func GetDatabaseStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                   stats.InUse,
		"idle":                     stats.Idle,
		"wait_count":               stats.WaitCount,
		"wait_duration":            stats.WaitDuration.String(),
		"max_idle_closed":          stats.MaxIdleClosed,
		"max_idle_time_closed":     stats.MaxIdleTimeClosed,
		"max_lifetime_closed":      stats.MaxLifetimeClosed,
	}, nil
}

// OptimizeDatabaseSettings applies MySQL-specific optimizations
func OptimizeDatabaseSettings(db *gorm.DB) error {
	log.Println("Applying MySQL optimization settings...")

	optimizations := []string{
		// Query cache optimization
		"SET SESSION query_cache_type = ON",
		"SET SESSION query_cache_size = 67108864", // 64MB
		
		// InnoDB optimizations
		"SET SESSION innodb_buffer_pool_size = 134217728", // 128MB
		"SET SESSION sort_buffer_size = 2097152",           // 2MB
		"SET SESSION read_buffer_size = 131072",            // 128KB
		"SET SESSION join_buffer_size = 262144",            // 256KB
		
		// Timeout settings
		"SET SESSION wait_timeout = 28800",                 // 8 hours
		"SET SESSION interactive_timeout = 28800",          // 8 hours
		
		// Character set
		"SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci",
	}

	for _, sql := range optimizations {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warning: Failed to apply optimization: %s - %v", sql, err)
		}
	}

	log.Println("Database optimization settings applied")
	return nil
}

// Helper functions for environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}