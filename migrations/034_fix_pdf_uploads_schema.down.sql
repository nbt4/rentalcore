-- Rollback: Remove columns added in 034
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS document_id;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS original_filename;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS stored_filename;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS file_hash;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS processing_status;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS processing_started_at;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS processing_completed_at;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS error_message;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS is_active;
ALTER TABLE pdf_uploads DROP COLUMN IF EXISTS uploaded_at;
