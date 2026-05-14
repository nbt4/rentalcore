ALTER TABLE jobs ADD COLUMN IF NOT EXISTS m365_event_id VARCHAR(255);

CREATE TABLE job_employees (
    job_id      BIGINT NOT NULL REFERENCES jobs(jobid) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    role        VARCHAR(100),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, employee_id)
);

CREATE INDEX idx_job_employees_job_id      ON job_employees(job_id);
CREATE INDEX idx_job_employees_employee_id ON job_employees(employee_id);
