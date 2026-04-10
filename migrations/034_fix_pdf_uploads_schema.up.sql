-- Migration 034: Align pdf_uploads table with Go model
-- The table was created with an older schema missing many columns.
-- This adds all missing columns and backfills data from existing columns.

-- Add document_id (the immediate error trigger)
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS document_id BIGINT DEFAULT NULL;

-- Add original_filename and stored_filename (backfill from existing filename column)
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS original_filename VARCHAR(500) DEFAULT '';
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS stored_filename VARCHAR(500) DEFAULT '';

UPDATE pdf_uploads
SET original_filename = COALESCE(filename, ''),
    stored_filename   = COALESCE(filename, '')
WHERE original_filename = '' OR stored_filename = '';

-- Add file_hash
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS file_hash VARCHAR(64) DEFAULT NULL;

-- Add processing_status (backfill from existing status column)
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS processing_status VARCHAR(50) DEFAULT 'pending';

UPDATE pdf_uploads
SET processing_status = COALESCE(status, 'pending')
WHERE processing_status = 'pending' AND status IS NOT NULL;

-- Add processing timestamps
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMP DEFAULT NULL;
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS processing_completed_at TIMESTAMP DEFAULT NULL;

-- Add error_message
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS error_message TEXT DEFAULT NULL;

-- Add is_active
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

-- Add uploaded_at (backfill from existing created_at column)
ALTER TABLE pdf_uploads ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

UPDATE pdf_uploads
SET uploaded_at = COALESCE(created_at, CURRENT_TIMESTAMP)
WHERE uploaded_at IS NULL OR uploaded_at = CURRENT_TIMESTAMP;
