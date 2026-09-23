package schema

import (
	"database/sql"
	"fmt"
)

// EnsureJobRequirementSources upgrades existing databases in place. Fresh
// installations already contain these columns in the umbrella init schema.
func EnsureJobRequirementSources(db *sql.DB) error {
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'job_product_requirements'
		  AND column_name = 'manual_quantity'
	)`).Scan(&exists); err != nil {
		return fmt.Errorf("check job requirement schema: %w", err)
	}
	if exists {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin job requirement migration: %w", err)
	}
	defer tx.Rollback()
	statements := []string{
		`ALTER TABLE job_product_requirements ADD COLUMN manual_quantity INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE job_product_requirements ADD COLUMN position_quantity INTEGER NOT NULL DEFAULT 0`,
		`WITH totals AS (
			SELECT job_id, product_id, SUM(GREATEST(1, ROUND(quantity)::integer)) AS quantity
			FROM job_positions WHERE position_type = 'product' AND product_id IS NOT NULL
			GROUP BY job_id, product_id
		)
		UPDATE job_product_requirements r
		SET position_quantity = COALESCE(t.quantity, 0),
		    manual_quantity = GREATEST(r.quantity - COALESCE(t.quantity, 0), 0),
		    quantity = GREATEST(r.quantity, COALESCE(t.quantity, 0))
		FROM totals t WHERE t.job_id = r.job_id AND t.product_id = r.product_id`,
		`UPDATE job_product_requirements SET manual_quantity = quantity
		 WHERE manual_quantity = 0 AND position_quantity = 0`,
		`INSERT INTO job_product_requirements (job_id, product_id, quantity, manual_quantity, position_quantity)
		 SELECT p.job_id, p.product_id, p.quantity, 0, p.quantity FROM (
			SELECT job_id, product_id, SUM(GREATEST(1, ROUND(quantity)::integer)) AS quantity
			FROM job_positions WHERE position_type = 'product' AND product_id IS NOT NULL
			GROUP BY job_id, product_id
		 ) p ON CONFLICT (job_id, product_id) DO NOTHING`,
		`ALTER TABLE job_product_requirements ADD CONSTRAINT chk_job_requirement_sources
		 CHECK (manual_quantity >= 0 AND position_quantity >= 0 AND quantity = manual_quantity + position_quantity)`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("apply job requirement migration: %w", err)
		}
	}
	return tx.Commit()
}
