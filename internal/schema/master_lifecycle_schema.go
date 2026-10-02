package schema

import "database/sql"

// RentalMasterLifecycleSQL is also shipped as umbrella migration 033.
const RentalMasterLifecycleSQL = `
CREATE TABLE IF NOT EXISTS venues(
 id SERIAL PRIMARY KEY,name VARCHAR(255) NOT NULL,street VARCHAR(255),house_number VARCHAR(50),zip VARCHAR(20),city VARCHAR(255),
 contact_name VARCHAR(255),phone VARCHAR(100),email VARCHAR(255),notes TEXT,
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS venue_id INTEGER REFERENCES venues(id) ON DELETE SET NULL;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE venues ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE venues ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP;
CREATE TABLE IF NOT EXISTS rental_mcp_mutation_receipts(
 id BIGSERIAL PRIMARY KEY,user_id INTEGER NOT NULL REFERENCES users(userid),
 operation VARCHAR(80) NOT NULL,key_hash CHAR(64) NOT NULL,request_hash CHAR(64) NOT NULL,
 response JSONB,status_code INTEGER NOT NULL DEFAULT 200,created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(user_id,operation,key_hash));
CREATE OR REPLACE FUNCTION rental_master_job_is_closed(value TEXT) RETURNS BOOLEAN AS $$
 SELECT lower(trim(COALESCE(value,''))) IN ('abgeschlossen','storniert','completed','paid','canceled','cancelled','abgerechnet');
$$ LANGUAGE SQL IMMUTABLE;
CREATE OR REPLACE FUNCTION rental_master_active_jobs(kind TEXT,record_id INTEGER) RETURNS JSONB AS $$
 SELECT COALESCE(jsonb_agg(jsonb_build_object('job_id',jobid,'status',status,'updated_at',updated_at) ORDER BY jobid),'[]'::jsonb)
 FROM (SELECT j.jobid,s.status,j.updated_at FROM jobs j LEFT JOIN status s ON s.statusid=j.statusid
 WHERE j.deleted_at IS NULL AND NOT rental_master_job_is_closed(s.status)
 AND ((kind='customer' AND j.customerid=record_id) OR (kind='venue' AND j.venue_id=record_id)) ORDER BY j.jobid LIMIT 1001) selected;
$$ LANGUAGE SQL STABLE;
CREATE OR REPLACE FUNCTION guard_rental_master_lifecycle() RETURNS TRIGGER AS $$
DECLARE old_business JSONB;new_business JSONB;record_id INTEGER;kind TEXT;
BEGIN
 kind:=CASE WHEN TG_TABLE_NAME='customers' THEN 'customer' ELSE 'venue' END;
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Archive rental masters to retain history' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN
  record_id:=CASE WHEN kind='customer' THEN (to_jsonb(OLD)->>'customerid')::int ELSE (to_jsonb(OLD)->>'id')::int END;
  old_business:=to_jsonb(OLD)-'updated_at'-'is_archived'-'archived_at'-'m365_id'-'m365_updated_at'-'gal_contact_id';
  new_business:=to_jsonb(NEW)-'updated_at'-'is_archived'-'archived_at'-'m365_id'-'m365_updated_at'-'gal_contact_id';
  IF (to_jsonb(NEW)->CASE WHEN kind='customer' THEN 'customerid' ELSE 'id' END) IS DISTINCT FROM (to_jsonb(OLD)->CASE WHEN kind='customer' THEN 'customerid' ELSE 'id' END) THEN
   RAISE EXCEPTION 'Rental master identity is immutable' USING ERRCODE='23514';END IF;
  IF COALESCE(NEW.is_archived,false) AND new_business IS DISTINCT FROM old_business THEN
   RAISE EXCEPTION 'Archive preserves rental master business fields' USING ERRCODE='23514';END IF;
  IF COALESCE(OLD.is_archived,false) AND (COALESCE(NEW.is_archived,false) OR new_business IS DISTINCT FROM old_business) THEN
   RAISE EXCEPTION 'Restore rental master before editing; restoration preserves business fields' USING ERRCODE='23514';END IF;
  IF NOT COALESCE(OLD.is_archived,false) AND COALESCE(NEW.is_archived,false) AND jsonb_array_length(rental_master_active_jobs(kind,record_id))>0 THEN
   RAISE EXCEPTION 'Active jobs block rental master archive' USING ERRCODE='23514';END IF;
  NEW.updated_at:=GREATEST(clock_timestamp() AT TIME ZONE 'UTC',OLD.updated_at+INTERVAL '1 microsecond');
 ELSE NEW.updated_at:=clock_timestamp() AT TIME ZONE 'UTC';END IF;
 NEW.archived_at:=CASE WHEN NEW.is_archived THEN CASE WHEN TG_OP='UPDATE' THEN COALESCE(OLD.archived_at,clock_timestamp() AT TIME ZONE 'UTC') ELSE clock_timestamp() AT TIME ZONE 'UTC' END ELSE NULL END;
 RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS customers_guard_lifecycle ON customers;
CREATE TRIGGER customers_guard_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON customers FOR EACH ROW EXECUTE FUNCTION guard_rental_master_lifecycle();
DROP TRIGGER IF EXISTS venues_guard_lifecycle ON venues;
CREATE TRIGGER venues_guard_lifecycle BEFORE INSERT OR UPDATE OR DELETE ON venues FOR EACH ROW EXECUTE FUNCTION guard_rental_master_lifecycle();
CREATE OR REPLACE FUNCTION guard_rental_job_masters() RETURNS TRIGGER AS $$
DECLARE status_name TEXT;
BEGIN
 IF TG_OP='UPDATE' AND (NEW.customerid,NEW.venue_id,NEW.statusid,NEW.deleted_at) IS NOT DISTINCT FROM (OLD.customerid,OLD.venue_id,OLD.statusid,OLD.deleted_at) THEN RETURN NEW;END IF;
 SELECT status INTO status_name FROM status WHERE statusid=NEW.statusid;
 IF NEW.deleted_at IS NULL AND NOT rental_master_job_is_closed(status_name) THEN
  PERFORM 1 FROM customers WHERE customerid=NEW.customerid AND COALESCE(is_archived,false)=false FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Active customer required for active jobs' USING ERRCODE='23514';END IF;
  IF NEW.venue_id IS NOT NULL THEN
   PERFORM 1 FROM venues WHERE id=NEW.venue_id AND is_archived=false FOR SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Active venue required for active jobs' USING ERRCODE='23514';END IF;
  END IF;
 END IF;RETURN NEW;
END;$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS jobs_guard_master_lifecycle ON jobs;
CREATE TRIGGER jobs_guard_master_lifecycle BEFORE INSERT OR UPDATE OF customerid,venue_id,statusid,deleted_at ON jobs FOR EACH ROW EXECUTE FUNCTION guard_rental_job_masters();
`

func EnsureRentalMasterLifecycle(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(RentalMasterLifecycleSQL); err != nil {
		return err
	}
	return tx.Commit()
}
