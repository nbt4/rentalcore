// Package repository provides PostgreSQL database connection for RentalCore
package repository

import (
	"fmt"
"time"

"go-barcode-webapp/internal/config"

"gorm.io/driver/postgres"
"gorm.io/gorm"
"gorm.io/gorm/logger"
"gorm.io/gorm/schema"

	applogger "go-barcode-webapp/internal/logger"
)

// Database wraps the GORM database connection
type Database struct {
*gorm.DB
}

// NewDatabase erstellt eine neue PostgreSQL-Datenbankverbindung
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
dsn := cfg.DSN()

var logLevel logger.LogLevel
if cfg.EnableQueryLogging {
logLevel = logger.Info
} else {
logLevel = logger.Warn
}

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
Logger:                 logger.Default.LogMode(logLevel),
PrepareStmt:            cfg.PrepareStmt,
SkipDefaultTransaction: false,
CreateBatchSize:        1000,
NamingStrategy: schema.NamingStrategy{
SingularTable: true,
},
DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKeyConstraintWhenMigrating,
})
if err != nil {
return nil, fmt.Errorf("failed to connect to database: %w", err)
}

sqlDB, err := db.DB()
if err != nil {
return nil, fmt.Errorf("failed to get sql.DB: %w", err)
}

// PostgreSQL unterstützt viele parallele Verbindungen
sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
sqlDB.SetConnMaxLifetime(time.Hour)
sqlDB.SetConnMaxIdleTime(30 * time.Minute)

applogger.LogInfo("PostgreSQL database connected: %s:%d/%s", cfg.Host, cfg.Port, cfg.Name)
return &Database{db}, nil
}

// Close schließt die Datenbankverbindung
func (db *Database) Close() error {
sqlDB, err := db.DB.DB()
if err != nil {
return err
}
return sqlDB.Close()
}

// Ping testet die Datenbankverbindung
func (db *Database) Ping() error {
sqlDB, err := db.DB.DB()
if err != nil {
return err
}
return sqlDB.Ping()
}
