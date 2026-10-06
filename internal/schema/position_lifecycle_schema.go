package schema

import (
	"database/sql"
	"fmt"
)

// RentalPositionLifecycleSQL is shared by startup and both migration inventories.
const RentalPositionLifecycleSQL = `ALTER TABLE job_positions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
DROP INDEX IF EXISTS idx_job_position_pdf_item;
CREATE UNIQUE INDEX idx_job_position_pdf_item ON job_positions(job_id,pdf_extraction_item_id) WHERE pdf_extraction_item_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_job_positions_active ON job_positions(job_id,product_id) WHERE deleted_at IS NULL;
CREATE OR REPLACE FUNCTION guard_rental_position_lifecycle() RETURNS TRIGGER AS $$
DECLARE old_business JSONB;new_business JSONB;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive job positions to retain identity and history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF (NEW.position_id,NEW.job_id,NEW.position_type,NEW.product_id,NEW.service_item_id,NEW.rental_equipment_id,NEW.created_at,NEW.pdf_extraction_item_id)
   IS DISTINCT FROM (OLD.position_id,OLD.job_id,OLD.position_type,OLD.product_id,OLD.service_item_id,OLD.rental_equipment_id,OLD.created_at,OLD.pdf_extraction_item_id) THEN
   RAISE EXCEPTION 'Position identity and provenance are immutable' USING ERRCODE='23514';END IF;
  old_business:=to_jsonb(OLD)-'updated_at'-'deleted_at';new_business:=to_jsonb(NEW)-'updated_at'-'deleted_at';
  IF OLD.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'Archived position is retained; create a reviewed new position' USING ERRCODE='23514';END IF;
  IF NEW.deleted_at IS NOT NULL THEN
   IF old_business IS DISTINCT FROM new_business THEN RAISE EXCEPTION 'Position archive preserves every business field' USING ERRCODE='23514';END IF;
   IF EXISTS(SELECT 1 FROM job_position_devices WHERE position_id=OLD.position_id) THEN RAISE EXCEPTION 'Assigned position devices block archive' USING ERRCODE='23514';END IF;
  END IF;
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',OLD.updated_at+INTERVAL '1 microsecond');
 ELSE
  IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'Create active positions' USING ERRCODE='23514';END IF;
  NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';
 END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_positions_guard_position_lifecycle ON job_positions;
CREATE TRIGGER job_positions_guard_position_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON job_positions FOR EACH ROW EXECUTE FUNCTION guard_rental_position_lifecycle();
CREATE OR REPLACE FUNCTION guard_rental_position_device_active() RETURNS TRIGGER AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM job_positions WHERE position_id=NEW.position_id AND deleted_at IS NULL FOR SHARE) THEN
  RAISE EXCEPTION 'Active existing position required for device assignment' USING ERRCODE='23514';END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_position_devices_guard_active_position ON job_position_devices;
CREATE TRIGGER job_position_devices_guard_active_position BEFORE INSERT OR UPDATE ON job_position_devices FOR EACH ROW EXECUTE FUNCTION guard_rental_position_device_active();
ALTER TABLE job_product_requirements ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE job_product_requirements ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_job_requirement_active ON job_product_requirements(job_id,product_id) WHERE deleted_at IS NULL;
CREATE OR REPLACE FUNCTION guard_rental_requirement_lifecycle() RETURNS TRIGGER AS $$
DECLARE parent_archived BOOLEAN;product_active BOOLEAN;position_count INTEGER:=0;assigned_count INTEGER:=0;old_business JSONB;new_business JSONB;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive material requirements to retain identity and history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF (NEW.id,NEW.job_id,NEW.product_id,NEW.created_at) IS DISTINCT FROM (OLD.id,OLD.job_id,OLD.product_id,OLD.created_at) THEN
   RAISE EXCEPTION 'Requirement identity and provenance are immutable' USING ERRCODE='23514';END IF;
  old_business:=to_jsonb(OLD)-'updated_at'-'deleted_at';new_business:=to_jsonb(NEW)-'updated_at'-'deleted_at';
  IF OLD.deleted_at IS NOT NULL AND (NEW.deleted_at IS NOT NULL OR new_business IS DISTINCT FROM old_business) THEN
   RAISE EXCEPTION 'Restore requirement before editing; restore preserves fields' USING ERRCODE='23514';END IF;
  NEW.updated_at:=GREATEST(clock_timestamp(),OLD.updated_at+INTERVAL '1 microsecond');
 ELSE
  IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'Create active requirements; archive is a separate reviewed workflow' USING ERRCODE='23514';END IF;
  NEW.updated_at:=clock_timestamp();
 END IF;
 SELECT COALESCE(to_jsonb(j)->>'deleted_at','')<>'' INTO parent_archived FROM jobs j WHERE jobid=NEW.job_id FOR SHARE;
 IF NOT FOUND OR parent_archived THEN RAISE EXCEPTION 'Existing non-archived job required' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' AND OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
  IF old_business IS DISTINCT FROM new_business THEN RAISE EXCEPTION 'Requirement archive preserves all quantities and fields' USING ERRCODE='23514';END IF;
  IF to_regclass('job_positions') IS NOT NULL THEN SELECT count(*) INTO position_count FROM job_positions WHERE job_id=NEW.job_id AND product_id=NEW.product_id AND position_type='product' AND COALESCE(to_jsonb(job_positions)->>'deleted_at','')='';END IF;
  IF to_regclass('job_devices') IS NOT NULL AND to_regclass('devices') IS NOT NULL THEN SELECT count(*) INTO assigned_count FROM job_devices jd JOIN devices d ON d.deviceid=jd.deviceid WHERE jd.jobid=NEW.job_id AND d.productid=NEW.product_id AND COALESCE(to_jsonb(jd)->>'pack_status','pending')<>'returned';END IF;
  IF position_count>0 OR assigned_count>0 THEN RAISE EXCEPTION 'Commercial positions or assigned equipment block requirement archive' USING ERRCODE='23514';END IF;
 ELSE
  IF to_regclass('products') IS NOT NULL THEN
   SELECT COALESCE(to_jsonb(p)->>'lifecycle_status','active')='active' INTO product_active FROM products p WHERE productid=NEW.product_id FOR SHARE;
   IF NOT FOUND OR NOT product_active THEN RAISE EXCEPTION 'Active existing requirement product required' USING ERRCODE='23514';END IF;
  END IF;
  IF to_regclass('job_devices') IS NOT NULL AND to_regclass('devices') IS NOT NULL THEN SELECT count(*) INTO assigned_count FROM job_devices jd JOIN devices d ON d.deviceid=jd.deviceid WHERE jd.jobid=NEW.job_id AND d.productid=NEW.product_id AND COALESCE(to_jsonb(jd)->>'pack_status','pending')<>'returned';END IF;
  IF NEW.quantity<assigned_count THEN RAISE EXCEPTION 'Requirement quantity cannot be below assigned equipment' USING ERRCODE='23514';END IF;
 END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_product_requirements_guard_requirement_lifecycle ON job_product_requirements;
CREATE TRIGGER job_product_requirements_guard_requirement_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON job_product_requirements FOR EACH ROW EXECUTE FUNCTION guard_rental_requirement_lifecycle();
`

func EnsureRentalPositionLifecycle(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(RentalPositionLifecycleSQL); err != nil {
		return fmt.Errorf("apply position lifecycle: %w", err)
	}
	return tx.Commit()
}
