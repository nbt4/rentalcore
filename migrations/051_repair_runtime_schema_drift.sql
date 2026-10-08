-- Repair PostgreSQL schema objects required by RentalCore at runtime.
-- The legacy MySQL migrations for these objects were never a valid PostgreSQL path.

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    object_type VARCHAR(100) NOT NULL,
    object_id VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL,
    username VARCHAR(255) NOT NULL,
    action TEXT NOT NULL,
    old_values TEXT,
    new_values TEXT,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    session_id VARCHAR(255),
    context TEXT,
    event_hash VARCHAR(64) NOT NULL UNIQUE,
    previous_hash VARCHAR(64),
    is_compliant BOOLEAN NOT NULL DEFAULT TRUE,
    retention_date TIMESTAMPTZ NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_object ON audit_events(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_user ON audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_session ON audit_events(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_previous_hash ON audit_events(previous_hash);
CREATE INDEX IF NOT EXISTS idx_audit_events_retention ON audit_events(retention_date);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);

CREATE TABLE IF NOT EXISTS gobd_records (
    id BIGSERIAL PRIMARY KEY,
    document_type VARCHAR(100) NOT NULL,
    document_id VARCHAR(255) NOT NULL,
    original_data TEXT NOT NULL,
    data_hash VARCHAR(64) NOT NULL,
    archive_date TIMESTAMPTZ NOT NULL,
    retention_date TIMESTAMPTZ NOT NULL,
    digital_sign TEXT,
    user_id BIGINT NOT NULL,
    company_id BIGINT NOT NULL,
    is_immutable BOOLEAN NOT NULL DEFAULT TRUE,
    archive_file_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gobd_records_document ON gobd_records(document_type, document_id);
CREATE INDEX IF NOT EXISTS idx_gobd_records_hash ON gobd_records(data_hash);
CREATE INDEX IF NOT EXISTS idx_gobd_records_archive_date ON gobd_records(archive_date);
CREATE INDEX IF NOT EXISTS idx_gobd_records_retention ON gobd_records(retention_date);
CREATE INDEX IF NOT EXISTS idx_gobd_records_user ON gobd_records(user_id);

CREATE TABLE IF NOT EXISTS retention_policies (
    id BIGSERIAL PRIMARY KEY,
    document_type VARCHAR(100) NOT NULL,
    retention_years INTEGER NOT NULL DEFAULT 10,
    legal_basis VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    auto_delete_after BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- A legacy compliance table used different column names. Preserve its rows if it
-- exists, while adding the columns used by the current Go models.
ALTER TABLE retention_policies ADD COLUMN IF NOT EXISTS retention_years INTEGER;
ALTER TABLE retention_policies ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE retention_policies ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE retention_policies ADD COLUMN IF NOT EXISTS auto_delete_after BOOLEAN DEFAULT FALSE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'retention_policies'
          AND column_name = 'retention_period_days'
    ) THEN
        EXECUTE 'UPDATE retention_policies
                 SET retention_years = GREATEST(1, CEIL(retention_period_days / 365.0)::INTEGER)
                 WHERE retention_years IS NULL';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'retention_policies'
          AND column_name = 'policy_description'
    ) THEN
        EXECUTE 'UPDATE retention_policies
                 SET description = policy_description
                 WHERE description IS NULL';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'retention_policies'
          AND column_name = 'auto_delete'
    ) THEN
        EXECUTE 'UPDATE retention_policies
                 SET auto_delete_after = COALESCE(auto_delete, FALSE)';
    END IF;
END $$;

UPDATE retention_policies SET retention_years = 10 WHERE retention_years IS NULL OR retention_years < 1;
UPDATE retention_policies SET is_active = TRUE WHERE is_active IS NULL;
UPDATE retention_policies SET auto_delete_after = FALSE WHERE auto_delete_after IS NULL;
ALTER TABLE retention_policies ALTER COLUMN retention_years SET DEFAULT 10;
ALTER TABLE retention_policies ALTER COLUMN retention_years SET NOT NULL;
ALTER TABLE retention_policies ALTER COLUMN is_active SET DEFAULT TRUE;
ALTER TABLE retention_policies ALTER COLUMN is_active SET NOT NULL;
ALTER TABLE retention_policies ALTER COLUMN auto_delete_after SET DEFAULT FALSE;
ALTER TABLE retention_policies ALTER COLUMN auto_delete_after SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_retention_policies_document_type
    ON retention_policies(document_type);

CREATE TABLE IF NOT EXISTS invoice_templates (
    template_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    html_template TEXT NOT NULL,
    css_styles TEXT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT REFERENCES users(userid) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_invoice_templates_default ON invoice_templates(is_default);
CREATE INDEX IF NOT EXISTS idx_invoice_templates_active ON invoice_templates(is_active);

ALTER TABLE pdf_extraction_items
    ADD COLUMN IF NOT EXISTS mapped_package_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_pdf_items_package
    ON pdf_extraction_items(mapped_package_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON kcu.constraint_schema = tc.constraint_schema
         AND kcu.constraint_name = tc.constraint_name
        JOIN information_schema.constraint_column_usage ccu
          ON ccu.constraint_schema = tc.constraint_schema
         AND ccu.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND tc.constraint_schema = current_schema()
          AND tc.table_name = 'pdf_extraction_items'
          AND kcu.column_name = 'mapped_package_id'
          AND ccu.table_schema = current_schema()
          AND ccu.table_name = 'product_packages'
          AND ccu.column_name = 'id'
    ) THEN
        ALTER TABLE pdf_extraction_items
            ADD CONSTRAINT fk_pdf_items_package
            FOREIGN KEY (mapped_package_id)
            REFERENCES product_packages(id)
            ON DELETE SET NULL;
    END IF;
END $$;
