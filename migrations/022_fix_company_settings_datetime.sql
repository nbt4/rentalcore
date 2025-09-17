-- Fix corrupted datetime values in company_settings table
UPDATE company_settings
SET created_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE created_at = '0000-00-00 00:00:00'
   OR updated_at = '0000-00-00 00:00:00'
   OR created_at IS NULL
   OR updated_at IS NULL;