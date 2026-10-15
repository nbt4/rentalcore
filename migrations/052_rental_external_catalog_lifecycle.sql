-- Retain catalog identities and serialize native/MCP references on the catalog row.
-- No data rewrite, new columns, stock movement or job price recalculation.
CREATE OR REPLACE FUNCTION rental_external_equipment_active_jobs(equipment INTEGER) RETURNS JSONB AS $$
 SELECT COALESCE(jsonb_agg(to_jsonb(selected) ORDER BY job_id),'[]'::jsonb)
 FROM (SELECT j.jobid AS job_id,j.job_code,s.status,j.updated_at
 FROM jobs j LEFT JOIN status s ON s.statusid=j.statusid
 WHERE j.deleted_at IS NULL AND NOT rental_master_job_is_closed(s.status) AND (
 EXISTS(SELECT 1 FROM job_positions p WHERE p.job_id=j.jobid AND p.position_type='rental' AND p.rental_equipment_id=equipment AND p.deleted_at IS NULL)
 OR EXISTS(SELECT 1 FROM job_rental_equipment r LEFT JOIN job_positions p ON p.position_id=r.position_id WHERE r.job_id=j.jobid AND r.equipment_id=equipment AND (r.position_id IS NULL OR p.deleted_at IS NULL)))
 ORDER BY j.jobid LIMIT 1001) selected;
$$ LANGUAGE SQL VOLATILE;

CREATE OR REPLACE FUNCTION guard_rental_external_equipment_catalog() RETURNS TRIGGER AS $$
DECLARE changed_identity BOOLEAN;old_business JSONB;new_business JSONB;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive external rental catalog records to retain identities and history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF (NEW.id,NEW.created_at) IS DISTINCT FROM (OLD.id,OLD.created_at) THEN RAISE EXCEPTION 'External rental identity and provenance are immutable' USING ERRCODE='23514';END IF;
  old_business:=to_jsonb(OLD)-'updated_at'-'is_active';new_business:=to_jsonb(NEW)-'updated_at'-'is_active';
  IF (NOT OLD.is_active OR NOT NEW.is_active) AND (old_business IS DISTINCT FROM new_business OR (NOT OLD.is_active AND NOT NEW.is_active)) THEN
   RAISE EXCEPTION 'Restore external rental before editing; archive and restore preserve every field' USING ERRCODE='23514';END IF;
  IF (OLD.is_active AND NOT NEW.is_active OR (NEW.name,NEW.supplier,NEW.supplier_id) IS DISTINCT FROM (OLD.name,OLD.supplier,OLD.supplier_id)) AND jsonb_array_length(rental_external_equipment_active_jobs(OLD.id))>0 THEN
   RAISE EXCEPTION 'Open jobs block external rental archival or supplier/identity changes' USING ERRCODE='23514';END IF;
  IF NOT OLD.is_active AND NEW.is_active AND NEW.supplier_id IS NOT NULL THEN
   PERFORM 1 FROM customers WHERE customerid=NEW.supplier_id AND is_supplier AND NOT COALESCE(is_archived,false) FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Active supplier required before external rental restoration' USING ERRCODE='23514';END IF;
  END IF;
  changed_identity:=(lower(trim(NEW.name)),lower(trim(COALESCE(NEW.supplier,'')))) IS DISTINCT FROM (lower(trim(OLD.name)),lower(trim(COALESCE(OLD.supplier,''))));
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',OLD.updated_at+INTERVAL '1 microsecond');
 ELSE
  IF NOT NEW.is_active THEN RAISE EXCEPTION 'Create active rental catalog records' USING ERRCODE='23514';END IF;
  changed_identity:=true;NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';
 END IF;
 IF changed_identity THEN
  PERFORM pg_advisory_xact_lock(hashtextextended(lower(trim(NEW.name))||chr(31)||lower(trim(COALESCE(NEW.supplier,''))),42852));
  IF EXISTS(SELECT 1 FROM rental_equipment WHERE id<>NEW.id AND lower(trim(name))=lower(trim(NEW.name)) AND lower(trim(COALESCE(supplier,'')))=lower(trim(COALESCE(NEW.supplier,'')))) THEN
   RAISE EXCEPTION 'Exact rental name/supplier identity already exists, including inactive records' USING ERRCODE='23505';END IF;
 END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS rental_equipment_guard_catalog_lifecycle ON rental_equipment;
CREATE TRIGGER rental_equipment_guard_catalog_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON rental_equipment FOR EACH ROW EXECUTE FUNCTION guard_rental_external_equipment_catalog();

CREATE OR REPLACE FUNCTION guard_rental_external_equipment_reference() RETURNS TRIGGER AS $$
DECLARE equipment INTEGER;job INTEGER;live BOOLEAN;
BEGIN
 IF TG_TABLE_NAME='job_positions' THEN
  IF NEW.position_type<>'rental' OR NEW.rental_equipment_id IS NULL OR NEW.deleted_at IS NOT NULL THEN RETURN NEW;END IF;
  IF TG_OP='UPDATE' AND (NEW.job_id,NEW.rental_equipment_id,NEW.deleted_at) IS NOT DISTINCT FROM (OLD.job_id,OLD.rental_equipment_id,OLD.deleted_at) THEN RETURN NEW;END IF;
  equipment:=NEW.rental_equipment_id;job:=NEW.job_id;
 ELSE
  IF TG_OP='UPDATE' AND (NEW.job_id,NEW.equipment_id,NEW.position_id) IS NOT DISTINCT FROM (OLD.job_id,OLD.equipment_id,OLD.position_id) THEN RETURN NEW;END IF;
  IF NEW.position_id IS NOT NULL AND EXISTS(SELECT 1 FROM job_positions WHERE position_id=NEW.position_id AND deleted_at IS NOT NULL) THEN RETURN NEW;END IF;
  equipment:=NEW.equipment_id;job:=NEW.job_id;
 END IF;
 SELECT j.deleted_at IS NULL AND NOT rental_master_job_is_closed(s.status) INTO live FROM jobs j LEFT JOIN status s ON s.statusid=j.statusid WHERE j.jobid=job FOR SHARE OF j;
 IF NOT FOUND THEN RAISE EXCEPTION 'Existing job required for external rental reference' USING ERRCODE='23514';END IF;
 IF live THEN
  PERFORM 1 FROM rental_equipment WHERE id=equipment AND is_active FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Active external rental catalog item required for open jobs' USING ERRCODE='23514';END IF;
 END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS job_positions_guard_external_catalog ON job_positions;
CREATE TRIGGER job_positions_guard_external_catalog BEFORE INSERT OR UPDATE ON job_positions FOR EACH ROW EXECUTE FUNCTION guard_rental_external_equipment_reference();
DROP TRIGGER IF EXISTS job_rental_equipment_guard_external_catalog ON job_rental_equipment;
CREATE TRIGGER job_rental_equipment_guard_external_catalog BEFORE INSERT OR UPDATE ON job_rental_equipment FOR EACH ROW EXECUTE FUNCTION guard_rental_external_equipment_reference();

CREATE OR REPLACE FUNCTION guard_rental_job_external_catalog() RETURNS TRIGGER AS $$
DECLARE equipment INTEGER;
BEGIN
 IF TG_OP='UPDATE' AND (NEW.statusid,NEW.deleted_at) IS NOT DISTINCT FROM (OLD.statusid,OLD.deleted_at) THEN RETURN NEW;END IF;
 IF NEW.deleted_at IS NOT NULL OR rental_master_job_is_closed((SELECT status FROM status WHERE statusid=NEW.statusid)) THEN RETURN NEW;END IF;
 FOR equipment IN SELECT p.rental_equipment_id FROM job_positions p WHERE p.job_id=NEW.jobid AND p.position_type='rental' AND p.deleted_at IS NULL AND p.rental_equipment_id IS NOT NULL
 UNION SELECT r.equipment_id FROM job_rental_equipment r LEFT JOIN job_positions p ON p.position_id=r.position_id WHERE r.job_id=NEW.jobid AND (r.position_id IS NULL OR p.deleted_at IS NULL) ORDER BY 1 LOOP
  PERFORM 1 FROM rental_equipment WHERE id=equipment AND is_active FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Restore external rental catalog before reopening its job' USING ERRCODE='23514';END IF;
 END LOOP;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS jobs_guard_external_catalog ON jobs;
CREATE TRIGGER jobs_guard_external_catalog BEFORE INSERT OR UPDATE OF statusid,deleted_at ON jobs FOR EACH ROW EXECUTE FUNCTION guard_rental_job_external_catalog();
