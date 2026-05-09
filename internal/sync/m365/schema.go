package m365

import (
	"database/sql"
	"log"
)

// EnsureCustomerM365Columns fügt die M365-Sync-Spalten zur customers-Tabelle hinzu
// wenn sie noch nicht existieren. Idempotent — safe bei jedem Start.
func EnsureCustomerM365Columns(db *sql.DB) error {
	columns := []struct {
		name       string
		definition string
	}{
		{"m365_id", "VARCHAR(255)"},
		{"m365_updated_at", "TIMESTAMP"},
		{"is_archived", "BOOLEAN NOT NULL DEFAULT FALSE"},
		{"archived_at", "TIMESTAMP"},
		{"updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
		{"created_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
	}

	for _, col := range columns {
		if err := ensureColumn(db, "customers", col.name, col.definition); err != nil {
			return err
		}
	}

	if err := ensureM365UniqueIndex(db); err != nil {
		log.Printf("M365 sync: warning — could not create m365_id index: %v", err)
	}

	log.Println("M365 sync: customer schema columns verified")
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
	`, table, column).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func ensureM365UniqueIndex(db *sql.DB) error {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pg_indexes
		WHERE tablename = 'customers' AND indexname = 'idx_customers_m365_id'
	`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX idx_customers_m365_id ON customers(m365_id) WHERE m365_id IS NOT NULL`)
	return err
}
