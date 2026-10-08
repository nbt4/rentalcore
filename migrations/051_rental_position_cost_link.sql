-- Commercial rental positions own revenue; the existing ledger retains supplier costs.
-- No legacy assignment is repaired automatically.
ALTER TABLE job_rental_equipment
    ADD COLUMN IF NOT EXISTS position_id BIGINT REFERENCES job_positions(position_id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS rental_unit_price NUMERIC;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='job_rental_equipment'::regclass AND conname='jre_linked_cost_snapshot') THEN
  ALTER TABLE job_rental_equipment ADD CONSTRAINT jre_linked_cost_snapshot CHECK(position_id IS NULL OR (rental_unit_price IS NOT NULL AND rental_unit_price>=0));
 END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_jre_position ON job_rental_equipment(position_id) WHERE position_id IS NOT NULL;

CREATE OR REPLACE FUNCTION validate_job_rental_position_link() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.position_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM job_positions p
        WHERE p.position_id = NEW.position_id AND p.job_id = NEW.job_id
          AND p.position_type = 'rental' AND p.rental_equipment_id = NEW.equipment_id AND p.quantity=NEW.quantity
    ) THEN
        RAISE EXCEPTION 'Supplier cost must reference its matching rental job position';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS validate_job_rental_position_link ON job_rental_equipment;
CREATE TRIGGER validate_job_rental_position_link BEFORE INSERT OR UPDATE OF position_id, job_id, equipment_id,quantity
    ON job_rental_equipment FOR EACH ROW EXECUTE FUNCTION validate_job_rental_position_link();

-- All owning write paths (including native job settings) retain the captured supplier price.
CREATE OR REPLACE FUNCTION sync_job_rental_day_costs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE job_rental_equipment c
    SET rental_unit_price = COALESCE(c.rental_unit_price, c.total_cost / NULLIF(c.quantity,0)::numeric /
            CASE WHEN OLD.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END),
        total_cost = round(COALESCE(c.rental_unit_price, c.total_cost / NULLIF(c.quantity,0)::numeric /
            CASE WHEN OLD.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END) * c.quantity *
            CASE WHEN NEW.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END, 2),
        updated_at = clock_timestamp()
    WHERE c.job_id = NEW.jobid AND c.quantity > 0;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS sync_job_rental_day_costs ON jobs;
CREATE TRIGGER sync_job_rental_day_costs AFTER UPDATE OF multiply_by_days ON jobs
    FOR EACH ROW WHEN (OLD.multiply_by_days IS DISTINCT FROM NEW.multiply_by_days)
    EXECUTE FUNCTION sync_job_rental_day_costs();

CREATE OR REPLACE FUNCTION sync_job_rental_position_costs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM job_rental_equipment c WHERE c.position_id=NEW.position_id
        AND (NEW.job_id<>c.job_id OR NEW.position_type<>'rental' OR NEW.rental_equipment_id IS DISTINCT FROM c.equipment_id)) THEN
        RAISE EXCEPTION 'Linked rental position identity cannot change';
    END IF;
    IF NEW.quantity IS DISTINCT FROM OLD.quantity THEN
        IF NEW.quantity < 1 OR NEW.quantity > 1000 OR NEW.quantity <> trunc(NEW.quantity) THEN
            RAISE EXCEPTION 'Linked rental quantity must be a whole number 1-1000';
        END IF;
        UPDATE job_rental_equipment c SET quantity=NEW.quantity,
            total_cost=round(c.rental_unit_price * NEW.quantity * CASE WHEN j.multiply_by_days THEN GREATEST(c.days_used,1) ELSE 1 END,2),
            updated_at=clock_timestamp()
        FROM jobs j WHERE c.position_id=NEW.position_id AND j.jobid=c.job_id AND c.rental_unit_price IS NOT NULL;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS sync_job_rental_position_costs ON job_positions;
CREATE TRIGGER sync_job_rental_position_costs AFTER UPDATE OF quantity,job_id,position_type,rental_equipment_id ON job_positions
    FOR EACH ROW WHEN (OLD.position_type='rental' OR NEW.position_type='rental')
    EXECUTE FUNCTION sync_job_rental_position_costs();
