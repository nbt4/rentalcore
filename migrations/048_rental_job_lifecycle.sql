CREATE TABLE IF NOT EXISTS skills (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT skills_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS employees (
    id           BIGSERIAL PRIMARY KEY,
    first_name   VARCHAR(100) NOT NULL,
    last_name    VARCHAR(100) NOT NULL,
    email        VARCHAR(255),
    phone        VARCHAR(50),
    mobile       VARCHAR(50),
    street       VARCHAR(255),
    house_number VARCHAR(20),
    zip          VARCHAR(20),
    city         VARCHAR(100),
    country      VARCHAR(100) NOT NULL DEFAULT 'Deutschland',
    date_of_birth DATE,
    iban         VARCHAR(50),
    notes        TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS employee_skills (
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id    BIGINT NOT NULL REFERENCES skills(id)    ON DELETE CASCADE,
    PRIMARY KEY (employee_id, skill_id)
);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS m365_event_id VARCHAR(255);

CREATE TABLE IF NOT EXISTS job_employees (
    job_id      BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    role        VARCHAR(100),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, employee_id)
);

CREATE INDEX IF NOT EXISTS idx_job_employees_job_id      ON job_employees(job_id);
CREATE INDEX IF NOT EXISTS idx_job_employees_employee_id ON job_employees(employee_id);

-- Preserve job identity/history and version every writer, including child workflows.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS revision INTEGER NOT NULL DEFAULT 1;
CREATE OR REPLACE FUNCTION rental_job_archive_blockers(jid INTEGER) RETURNS JSONB AS $$
DECLARE issued INTEGER:=0;open_tasks INTEGER:=0;active_cases INTEGER:=0;
BEGIN
 IF to_regclass('job_devices') IS NOT NULL THEN SELECT count(*) INTO issued FROM job_devices WHERE jobid=jid AND pack_status='issued';END IF;
 IF to_regclass('warehouse_tasks') IS NOT NULL THEN SELECT count(*) INTO open_tasks FROM warehouse_tasks WHERE job_id=jid AND status IN ('open','in_progress') AND NOT is_archived;END IF;
 IF to_regclass('cases') IS NOT NULL AND EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='cases' AND column_name='current_job_id') THEN
  SELECT count(*) INTO active_cases FROM cases c WHERE current_job_id=jid AND COALESCE(to_jsonb(c)->>'lifecycle_status','active')='active';
 END IF;
 RETURN jsonb_build_object('issued_devices',issued,'open_warehouse_tasks',open_tasks,'active_cases',active_cases);
END;$$ LANGUAGE plpgsql STABLE;
CREATE OR REPLACE FUNCTION guard_rental_job_lifecycle() RETURNS TRIGGER AS $$
DECLARE old_business JSONB;new_business JSONB;b JSONB;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive jobs to retain business history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  IF NEW.jobid IS DISTINCT FROM OLD.jobid THEN RAISE EXCEPTION 'Job identity is immutable' USING ERRCODE='23514';END IF;
  old_business:=to_jsonb(OLD)-'updated_at'-'updated_by'-'revision'-'m365_event_id'-'deleted_at';
  new_business:=to_jsonb(NEW)-'updated_at'-'updated_by'-'revision'-'m365_event_id'-'deleted_at';
  IF OLD.deleted_at IS NOT NULL AND new_business IS DISTINCT FROM old_business THEN
   RAISE EXCEPTION 'Restore job before changing business fields' USING ERRCODE='23514';END IF;
  IF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS DISTINCT FROM OLD.deleted_at AND NEW.deleted_at IS NOT NULL THEN
   RAISE EXCEPTION 'Archived job timestamp is retained' USING ERRCODE='23514';END IF;
  IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
   IF (new_business-'statusid') IS DISTINCT FROM (old_business-'statusid') OR (NEW.statusid IS DISTINCT FROM OLD.statusid AND NOT (OLD.statusid IN (1,2) AND NEW.statusid=6)) THEN
    RAISE EXCEPTION 'Job archive preserves fields and may only close an open draft' USING ERRCODE='23514';END IF;
   b:=rental_job_archive_blockers(OLD.jobid);
   IF (b->>'issued_devices')::int+(b->>'open_warehouse_tasks')::int+(b->>'active_cases')::int>0 THEN
    RAISE EXCEPTION 'Active warehouse work or issued equipment blocks job archive' USING ERRCODE='23514';END IF;
  END IF;
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',COALESCE(OLD.updated_at,clock_timestamp() AT TIME ZONE 'UTC')+INTERVAL '1 microsecond');
 ELSE NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';END IF;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS jobs_guard_lifecycle_version ON jobs;
CREATE TRIGGER jobs_guard_lifecycle_version BEFORE INSERT OR UPDATE OR DELETE ON jobs FOR EACH ROW EXECUTE FUNCTION guard_rental_job_lifecycle();
CREATE OR REPLACE FUNCTION rental_job_child_owner(row_data JSONB,key_name TEXT) RETURNS INTEGER AS $$
DECLARE jid INTEGER;
BEGIN
 IF key_name='position_id' THEN SELECT job_id INTO jid FROM job_positions WHERE position_id=(row_data->>key_name)::bigint;
 ELSIF key_name='job_package_id' THEN SELECT job_id INTO jid FROM job_packages WHERE job_package_id=(row_data->>key_name)::bigint;
 ELSE jid:=(row_data->>key_name)::int;END IF;
 RETURN jid;
END;$$ LANGUAGE plpgsql STABLE;
CREATE OR REPLACE FUNCTION guard_rental_job_child() RETURNS TRIGGER AS $$
DECLARE old_id INTEGER;new_id INTEGER;archived TIMESTAMP;old_row JSONB;new_row JSONB;
BEGIN
 IF TG_OP<>'INSERT' THEN old_row:=to_jsonb(OLD);old_id:=rental_job_child_owner(old_row,TG_ARGV[0]);END IF;
 IF TG_OP<>'DELETE' THEN new_row:=to_jsonb(NEW);new_id:=rental_job_child_owner(new_row,TG_ARGV[0]);END IF;
 FOR archived IN SELECT deleted_at FROM jobs WHERE jobid IN (old_id,new_id) FOR SHARE LOOP
  IF archived IS NOT NULL AND NOT (TG_OP='UPDATE' AND old_id IS NOT DISTINCT FROM new_id AND
   (old_row-'updated_at'-'m365_event_id') IS NOT DISTINCT FROM (new_row-'updated_at'-'m365_event_id')) THEN
   RAISE EXCEPTION 'Archived job contents are retained; restore before editing' USING ERRCODE='23514';END IF;
 END LOOP;
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END;$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION touch_rental_job_child_version() RETURNS TRIGGER AS $$
DECLARE old_id INTEGER;new_id INTEGER;
BEGIN
 IF TG_OP<>'INSERT' THEN old_id:=rental_job_child_owner(to_jsonb(OLD),TG_ARGV[0]);END IF;
 IF TG_OP<>'DELETE' THEN new_id:=rental_job_child_owner(to_jsonb(NEW),TG_ARGV[0]);END IF;
 UPDATE jobs SET updated_at=clock_timestamp() AT TIME ZONE 'UTC' WHERE jobid IN (old_id,new_id);
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END;$$ LANGUAGE plpgsql;
DO $$ DECLARE item TEXT;key_name TEXT;BEGIN
 FOREACH item IN ARRAY ARRAY['job_devices','job_product_requirements','job_packages','job_positions','job_rental_equipment','job_employees','job_attachments','job_position_devices','job_package_reservations'] LOOP
  IF to_regclass(item) IS NOT NULL THEN
   key_name:=CASE WHEN item='job_devices' THEN 'jobid' WHEN item='job_position_devices' THEN 'position_id' WHEN item='job_package_reservations' THEN 'job_package_id' ELSE 'job_id' END;
   EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I',item||'_guard_job_lifecycle',item);
   EXECUTE format('CREATE TRIGGER %I BEFORE INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION guard_rental_job_child(%L)',item||'_guard_job_lifecycle',item,key_name);
   EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I',item||'_touch_job_version',item);
   EXECUTE format('CREATE TRIGGER %I AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION touch_rental_job_child_version(%L)',item||'_touch_job_version',item,key_name);
  END IF;
 END LOOP;
END;$$;
