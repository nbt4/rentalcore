-- GoBD and GDPR Compliance Tables Migration
-- This migration creates all necessary tables for legal compliance

-- Table: audit_logs (GoBD-compliant audit trail)
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    entity_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(50) NOT NULL,
    user_id BIGINT UNSIGNED,
    changes JSON,
    metadata JSON,
    hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_audit_logs_entity (entity_type, entity_id),
    INDEX idx_audit_logs_user (user_id),
    INDEX idx_audit_logs_timestamp (timestamp),
    INDEX idx_audit_logs_hash (hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: archived_documents (GoBD-compliant document archiving)
CREATE TABLE IF NOT EXISTS archived_documents (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    document_type VARCHAR(100) NOT NULL,
    document_id BIGINT UNSIGNED NOT NULL,
    original_hash VARCHAR(64) NOT NULL,
    archived_data LONGTEXT NOT NULL,
    archived_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retention_until TIMESTAMP NOT NULL,
    legal_basis VARCHAR(200) NOT NULL,
    archive_format VARCHAR(50) NOT NULL DEFAULT 'json',
    compression_used BOOLEAN DEFAULT FALSE,
    encryption_used BOOLEAN DEFAULT FALSE,
    archive_path VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_document (document_type, document_id),
    INDEX idx_archived_docs_type (document_type),
    INDEX idx_archived_docs_retention (retention_until),
    INDEX idx_archived_docs_hash (original_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: document_signatures (Digital signatures for invoice integrity)
CREATE TABLE IF NOT EXISTS document_signatures (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    document_type VARCHAR(100) NOT NULL,
    document_id BIGINT UNSIGNED NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    signature_data TEXT NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'RSA-SHA256',
    public_key TEXT NOT NULL,
    signed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    signer_id BIGINT UNSIGNED,
    verification_status ENUM('valid', 'invalid', 'pending') DEFAULT 'pending',
    last_verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_document_signature (document_type, document_id),
    INDEX idx_doc_signatures_type (document_type),
    INDEX idx_doc_signatures_signer (signer_id),
    INDEX idx_doc_signatures_status (verification_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: retention_policies (Data retention policy definitions)
CREATE TABLE IF NOT EXISTS retention_policies (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    data_type VARCHAR(100) NOT NULL,
    retention_period_days INT UNSIGNED NOT NULL,
    legal_basis VARCHAR(200) NOT NULL,
    auto_delete BOOLEAN DEFAULT FALSE,
    policy_description TEXT,
    effective_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    effective_until TIMESTAMP NULL,
    created_by BIGINT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_active_policy (data_type, effective_until),
    INDEX idx_retention_policies_type (data_type),
    INDEX idx_retention_policies_effective (effective_from, effective_until)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: consent_records (GDPR consent tracking)
CREATE TABLE IF NOT EXISTS consent_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    data_type VARCHAR(100) NOT NULL,
    purpose VARCHAR(200) NOT NULL,
    consent_given BOOLEAN NOT NULL,
    consent_date TIMESTAMP NOT NULL,
    expiry_date TIMESTAMP NULL,
    legal_basis VARCHAR(100) NOT NULL,
    withdrawn_at TIMESTAMP NULL,
    version VARCHAR(20) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_consent_user (user_id),
    INDEX idx_consent_type_purpose (data_type, purpose),
    INDEX idx_consent_date (consent_date),
    INDEX idx_consent_expiry (expiry_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: data_processing_records (GDPR processing activity records)
CREATE TABLE IF NOT EXISTS data_processing_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    data_type VARCHAR(100) NOT NULL,
    processing_type VARCHAR(100) NOT NULL,
    purpose VARCHAR(200) NOT NULL,
    legal_basis VARCHAR(100) NOT NULL,
    data_controller VARCHAR(200) NOT NULL,
    data_processor VARCHAR(200) NULL,
    recipients JSON,
    transfer_country VARCHAR(2) NULL,
    retention_period VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_data_processing_user (user_id),
    INDEX idx_data_processing_type (data_type),
    INDEX idx_data_processing_purpose (purpose),
    INDEX idx_data_processing_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: data_subject_requests (GDPR data subject requests)
CREATE TABLE IF NOT EXISTS data_subject_requests (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    request_type ENUM('access', 'rectification', 'erasure', 'portability', 'restriction', 'objection') NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'rejected') NOT NULL DEFAULT 'pending',
    description TEXT,
    requested_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    processor_id BIGINT UNSIGNED NULL,
    response TEXT,
    response_data LONGTEXT,
    verification TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_data_subject_user (user_id),
    INDEX idx_data_subject_type (request_type),
    INDEX idx_data_subject_status (status),
    INDEX idx_data_subject_requested (requested_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Table: encrypted_personal_data (GDPR-compliant encrypted personal data)
CREATE TABLE IF NOT EXISTS encrypted_personal_data (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    data_type VARCHAR(100) NOT NULL,
    encrypted_data LONGTEXT NOT NULL,
    key_version VARCHAR(20) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_user_data_type (user_id, data_type),
    INDEX idx_encrypted_data_user (user_id),
    INDEX idx_encrypted_data_type (data_type),
    INDEX idx_encrypted_data_key_version (key_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default retention policies (German legal requirements)
INSERT INTO retention_policies (data_type, retention_period_days, legal_basis, auto_delete, policy_description) VALUES
('invoice_data', 3650, 'HGB §257, AO §147', TRUE, 'Handelsrechtliche und steuerrechtliche Aufbewahrung von Rechnungsdaten - 10 Jahre'),
('customer_data', 3650, 'HGB §257, AO §147', FALSE, 'Geschäftsbriefe und Handelsbücher - 10 Jahre'),
('payment_data', 2190, 'HGB §257', TRUE, 'Zahlungsbelege und Kontoauszüge - 6 Jahre'),
('contract_data', 3650, 'BGB §195ff', FALSE, 'Vertragsunterlagen - 10 Jahre (Gewährleistung)'),
('tax_data', 3650, 'AO §147', TRUE, 'Steuerrelevante Unterlagen - 10 Jahre'),
('employee_data', 1095, 'DSGVO Art. 5', FALSE, 'Personalunterlagen - 3 Jahre nach Beendigung'),
('marketing_consent', 1095, 'DSGVO Art. 7', TRUE, 'Marketing-Einwilligungen - 3 Jahre'),
('access_logs', 2190, 'DSGVO Art. 32', TRUE, 'Zugriffsprotokolle - 6 Jahre'),
('backup_data', 365, 'DSGVO Art. 32', TRUE, 'Backup-Daten - 1 Jahr');

-- Create indexes for better performance
CREATE INDEX idx_audit_logs_chain ON audit_logs(previous_hash, hash);
CREATE INDEX idx_archived_documents_legal ON archived_documents(legal_basis);
CREATE INDEX idx_document_signatures_hash ON document_signatures(content_hash);

-- Add foreign key constraints (assuming standard user table exists)
-- ALTER TABLE consent_records ADD CONSTRAINT fk_consent_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- ALTER TABLE data_processing_records ADD CONSTRAINT fk_processing_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- ALTER TABLE data_subject_requests ADD CONSTRAINT fk_subject_request_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- ALTER TABLE encrypted_personal_data ADD CONSTRAINT fk_encrypted_data_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;