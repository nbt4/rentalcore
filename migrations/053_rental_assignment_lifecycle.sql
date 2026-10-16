-- Retain linked rental assignments and restore only reviewed rental positions.
CREATE OR REPLACE FUNCTION guard_rental_position_lifecycle() RETURNS TRIGGER AS $$
DECLARE old_business JSONB;new_business JSONB;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive job positions to retain identity and history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF (NEW.position_id,NEW.job_id,NEW.position_type,NEW.product_id,NEW.service_item_id,NEW.rental_equipment_id,NEW.created_at,NEW.pdf_extraction_item_id)
   IS DISTINCT FROM (OLD.position_id,OLD.job_id,OLD.position_type,OLD.product_id,OLD.service_item_id,OLD.rental_equipment_id,OLD.created_at,OLD.pdf_extraction_item_id) THEN
   RAISE EXCEPTION 'Position identity and provenance are immutable' USING ERRCODE='23514';END IF;
  old_business:=to_jsonb(OLD)-'updated_at'-'deleted_at';new_business:=to_jsonb(NEW)-'updated_at'-'deleted_at';
  IF OLD.deleted_at IS NOT NULL THEN
   IF OLD.position_type<>'rental' OR NEW.deleted_at IS NOT NULL OR old_business IS DISTINCT FROM new_business THEN
    RAISE EXCEPTION 'Restore a retained rental position without changing business fields' USING ERRCODE='23514';END IF;
   IF NOT EXISTS(SELECT 1 FROM job_rental_equipment WHERE position_id=OLD.position_id AND job_id=OLD.job_id AND equipment_id=OLD.rental_equipment_id) THEN
    RAISE EXCEPTION 'Matching retained supplier assignment required for restore' USING ERRCODE='23514';END IF;
   PERFORM 1 FROM rental_equipment WHERE id=NEW.rental_equipment_id AND is_active FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Restore rental catalog first' USING ERRCODE='23514';END IF;
  END IF;
  IF NEW.deleted_at IS NOT NULL THEN
   IF old_business IS DISTINCT FROM new_business THEN RAISE EXCEPTION 'Position archive preserves every business field' USING ERRCODE='23514';END IF;
   IF EXISTS(SELECT 1 FROM job_position_devices WHERE position_id=OLD.position_id) THEN RAISE EXCEPTION 'Assigned position devices block archive' USING ERRCODE='23514';END IF;
  END IF;
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',OLD.updated_at+INTERVAL '1 microsecond');
 ELSE
  IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'Create active positions' USING ERRCODE='23514';END IF;
  NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';
 END IF;
 IF NEW.position_type='rental' AND NEW.deleted_at IS NULL AND
  (TG_OP='INSERT' OR OLD.deleted_at IS NOT NULL) AND EXISTS(SELECT 1 FROM job_positions p WHERE p.job_id=NEW.job_id AND p.rental_equipment_id=NEW.rental_equipment_id AND p.position_type='rental' AND p.deleted_at IS NULL AND p.position_id<>NEW.position_id) THEN
  RAISE EXCEPTION 'An active rental position already exists for this job and equipment' USING ERRCODE='23514';END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_positions_guard_position_lifecycle ON job_positions;
CREATE TRIGGER job_positions_guard_position_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON job_positions FOR EACH ROW EXECUTE FUNCTION guard_rental_position_lifecycle();

CREATE OR REPLACE FUNCTION guard_rental_assignment_history() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE archived BOOLEAN;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive the linked rental position to retain supplier history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF (NEW.job_id,NEW.equipment_id,NEW.created_at) IS DISTINCT FROM (OLD.job_id,OLD.equipment_id,OLD.created_at) OR (OLD.position_id IS NOT NULL AND NEW.position_id IS DISTINCT FROM OLD.position_id) THEN
   RAISE EXCEPTION 'Supplier assignment identity and linkage are immutable' USING ERRCODE='23514';END IF;
 END IF;
 IF NEW.position_id IS NOT NULL THEN
  SELECT deleted_at IS NOT NULL INTO archived FROM job_positions WHERE position_id=NEW.position_id FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Matching rental position required' USING ERRCODE='23514';END IF;
  IF archived THEN
   IF TG_OP='INSERT' OR (to_jsonb(NEW)-'updated_at') IS DISTINCT FROM (to_jsonb(OLD)-'updated_at') THEN
    RAISE EXCEPTION 'Archived rental supplier snapshots are retained' USING ERRCODE='23514';END IF;
   NEW.updated_at:=OLD.updated_at;
  ELSE
   IF NEW.quantity<1 OR NEW.quantity>1000 OR NEW.days_used<1 OR NEW.days_used>365 OR NEW.rental_unit_price<0 THEN
    RAISE EXCEPTION 'Valid quantity, days and captured supplier price required' USING ERRCODE='23514';END IF;
   IF TG_OP='UPDATE' THEN NEW.updated_at:=GREATEST(clock_timestamp(),OLD.updated_at+INTERVAL '1 microsecond');
   ELSE NEW.updated_at:=clock_timestamp();END IF;
  END IF;
 END IF;
 RETURN NEW;
END;$$;
DROP TRIGGER IF EXISTS job_rental_equipment_guard_history ON job_rental_equipment;
CREATE TRIGGER job_rental_equipment_guard_history BEFORE INSERT OR UPDATE OR DELETE ON job_rental_equipment FOR EACH ROW EXECUTE FUNCTION guard_rental_assignment_history();
CREATE OR REPLACE FUNCTION sync_job_rental_day_costs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE job_rental_equipment c
    SET rental_unit_price = COALESCE(c.rental_unit_price, c.total_cost / NULLIF(c.quantity,0)::numeric /
            CASE WHEN OLD.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END),
        total_cost = round(COALESCE(c.rental_unit_price, c.total_cost / NULLIF(c.quantity,0)::numeric /
            CASE WHEN OLD.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END) * c.quantity *
            CASE WHEN NEW.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END, 2),
        updated_at = clock_timestamp()
    WHERE c.job_id = NEW.jobid AND c.quantity > 0 AND (c.position_id IS NULL OR EXISTS(SELECT 1 FROM job_positions p WHERE p.position_id=c.position_id AND p.deleted_at IS NULL));
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION normalize_job_rental_captured_cost() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE multiply_days BOOLEAN;
BEGIN
    IF NEW.position_id IS NOT NULL AND EXISTS(SELECT 1 FROM job_positions WHERE position_id=NEW.position_id AND deleted_at IS NOT NULL) THEN RETURN NEW;END IF;
    IF NEW.rental_unit_price IS NOT NULL THEN
        SELECT COALESCE(multiply_by_days,true) INTO multiply_days FROM jobs WHERE jobid=NEW.job_id;
        NEW.total_cost=round(NEW.rental_unit_price * NEW.quantity *
            CASE WHEN multiply_days THEN GREATEST(NEW.days_used,1) ELSE 1 END,2);
    END IF;
    RETURN NEW;
END;
$$;
