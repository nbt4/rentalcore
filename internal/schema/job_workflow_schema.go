package schema

import (
	"database/sql"
	"fmt"
)

// EnsureJobWorkflowColumns keeps installations created before the job workflow
// revision and PDF position provenance compatible with the current model.
func EnsureJobWorkflowColumns(db *sql.DB) error {
	statements := []string{
		"ALTER TABLE jobs ADD COLUMN IF NOT EXISTS revision INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE job_positions ADD COLUMN IF NOT EXISTS pdf_extraction_item_id BIGINT",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_job_position_pdf_item ON job_positions(job_id, pdf_extraction_item_id) WHERE pdf_extraction_item_id IS NOT NULL",
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("ensure job workflow schema: %w", err)
		}
	}
	return nil
}
