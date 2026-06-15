package schema

import (
	"database/sql"

	"go-barcode-webapp/internal/logger"
)

func EnsureVenuesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS venues (
			id           SERIAL PRIMARY KEY,
			name         VARCHAR(255) NOT NULL,
			street       VARCHAR(255),
			house_number VARCHAR(50),
			zip          VARCHAR(20),
			city         VARCHAR(255),
			contact_name VARCHAR(255),
			phone        VARCHAR(100),
			email        VARCHAR(255),
			notes        TEXT,
			created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	logger.LogInfo("venues schema: table verified")
	return nil
}

func EnsureJobsVenueID(db *sql.DB) error {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'jobs' AND column_name = 'venue_id'
	`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec(`ALTER TABLE jobs ADD COLUMN venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL`)
	if err != nil {
		return err
	}
	logger.LogInfo("venues schema: venue_id column added to jobs")
	return nil
}
