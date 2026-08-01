-- Create the document management tables for PostgreSQL installations.
-- The original migration 002 uses MySQL-only ENUM/AUTO_INCREMENT syntax.

CREATE TABLE IF NOT EXISTS documents (
    documentid BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('job', 'device', 'customer', 'user', 'system')),
    entity_id VARCHAR(50) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    document_type VARCHAR(20) NOT NULL CHECK (document_type IN ('contract', 'manual', 'photo', 'invoice', 'receipt', 'signature', 'other')),
    description TEXT,
    uploaded_by BIGINT REFERENCES users(userid) ON DELETE SET NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,
    parent_documentid BIGINT REFERENCES documents(documentid) ON DELETE SET NULL,
    checksum VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_documents_entity_type
    ON documents(entity_type, entity_id, document_type);
CREATE INDEX IF NOT EXISTS idx_documents_uploaded_date
    ON documents(uploaded_at, document_type);
CREATE INDEX IF NOT EXISTS idx_documents_filename
    ON documents(filename);

CREATE TABLE IF NOT EXISTS digital_signatures (
    signatureid BIGSERIAL PRIMARY KEY,
    documentid BIGINT NOT NULL REFERENCES documents(documentid) ON DELETE CASCADE,
    signer_name VARCHAR(100) NOT NULL,
    signer_email VARCHAR(100),
    signer_role VARCHAR(50),
    signature_data TEXT NOT NULL,
    signed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    verification_code VARCHAR(100),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_digital_signatures_document_signer
    ON digital_signatures(documentid, signer_email);
CREATE INDEX IF NOT EXISTS idx_digital_signatures_signed_date
    ON digital_signatures(signed_at);
