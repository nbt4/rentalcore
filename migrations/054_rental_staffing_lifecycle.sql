-- Retain staffing identity/history and version every native/MCP writer.
ALTER TABLE job_employees ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE job_employees ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE OR REPLACE FUNCTION guard_rental_staffing_lifecycle() RETURNS TRIGGER AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive staffing to retain history; deactivate referenced employees instead of deleting them' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF NEW.job_id IS DISTINCT FROM OLD.job_id OR NEW.employee_id IS DISTINCT FROM OLD.employee_id THEN
   RAISE EXCEPTION 'Staffing identity is immutable' USING ERRCODE='23514';END IF;
  IF OLD.deleted_at IS NOT NULL AND (to_jsonb(NEW)-'updated_at'-'m365_event_id'-'deleted_at') IS DISTINCT FROM (to_jsonb(OLD)-'updated_at'-'m365_event_id'-'deleted_at') THEN
   RAISE EXCEPTION 'Restore staffing before editing fields' USING ERRCODE='23514';END IF;
  IF NEW.deleted_at IS DISTINCT FROM OLD.deleted_at AND NEW.role IS DISTINCT FROM OLD.role THEN
   RAISE EXCEPTION 'Staffing lifecycle preserves role' USING ERRCODE='23514';END IF;
  IF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NOT NULL AND NEW.deleted_at IS DISTINCT FROM OLD.deleted_at THEN
   RAISE EXCEPTION 'Retain original archive timestamp' USING ERRCODE='23514';END IF;
  IF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
   PERFORM id FROM employees WHERE id=NEW.employee_id AND is_active FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Activate employee before restoring staffing' USING ERRCODE='23514';END IF;
  END IF;
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',OLD.updated_at+INTERVAL '1 microsecond');
 ELSE
  PERFORM id FROM employees WHERE id=NEW.employee_id AND is_active FOR SHARE;
  IF NEW.deleted_at IS NOT NULL OR NOT FOUND THEN
   RAISE EXCEPTION 'New staffing requires active employee and active assignment' USING ERRCODE='23514';END IF;
  NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';
 END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_employees_guard_staffing_lifecycle ON job_employees;
CREATE TRIGGER job_employees_guard_staffing_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON job_employees FOR EACH ROW EXECUTE FUNCTION guard_rental_staffing_lifecycle();
