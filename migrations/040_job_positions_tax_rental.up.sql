-- 040_job_positions_tax_rental.up.sql
-- Add tax_rate per position (default 19%) and rental_equipment reference

ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS tax_rate DECIMAL(5,2) NOT NULL DEFAULT 19.00,
  ADD COLUMN IF NOT EXISTS rental_equipment_id INT REFERENCES rental_equipment(id) ON DELETE SET NULL;

ALTER TABLE job_positions
  DROP CONSTRAINT IF EXISTS job_positions_position_type_check;

ALTER TABLE job_positions
  ADD CONSTRAINT job_positions_position_type_check
  CHECK (position_type IN ('product', 'service', 'rental', 'package'));

CREATE INDEX IF NOT EXISTS idx_job_positions_rental_equipment ON job_positions(rental_equipment_id);
