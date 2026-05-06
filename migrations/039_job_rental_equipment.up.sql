-- 039_job_rental_equipment.up.sql
CREATE TABLE IF NOT EXISTS job_rental_equipment (
    job_id       BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    equipment_id INT    NOT NULL REFERENCES rental_equipment(id) ON DELETE CASCADE,
    quantity     INT    NOT NULL DEFAULT 1,
    days_used    INT    NOT NULL DEFAULT 1,
    total_cost   DECIMAL(12,2) NOT NULL DEFAULT 0,
    notes        VARCHAR(500) DEFAULT '',
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (job_id, equipment_id)
);
CREATE INDEX IF NOT EXISTS idx_jre_job ON job_rental_equipment(job_id);
CREATE INDEX IF NOT EXISTS idx_jre_equipment ON job_rental_equipment(equipment_id);
