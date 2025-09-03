-- Migration: Add German Business Fields to Company Settings
-- Description: Adds banking, legal, and invoice text fields for German compliance (GoBD)
-- Date: 2025-06-20

-- Add German banking fields
ALTER TABLE `company_settings` 
ADD COLUMN `bank_name` VARCHAR(255) NULL COMMENT 'Name der Bank für Rechnungen',
ADD COLUMN `iban` VARCHAR(34) NULL COMMENT 'IBAN für Zahlungen',
ADD COLUMN `bic` VARCHAR(11) NULL COMMENT 'BIC/SWIFT Code',
ADD COLUMN `account_holder` VARCHAR(255) NULL COMMENT 'Kontoinhaber falls abweichend';

-- Add German legal fields
ALTER TABLE `company_settings` 
ADD COLUMN `ceo_name` VARCHAR(255) NULL COMMENT 'Name des Geschäftsführers',
ADD COLUMN `register_court` VARCHAR(255) NULL COMMENT 'Registergericht',
ADD COLUMN `register_number` VARCHAR(100) NULL COMMENT 'Handelsregisternummer';

-- Add invoice text fields
ALTER TABLE `company_settings` 
ADD COLUMN `footer_text` TEXT NULL COMMENT 'Footer-Text für Rechnungen',
ADD COLUMN `payment_terms_text` TEXT NULL COMMENT 'Zahlungsbedingungen Text';

-- Update existing record with German defaults if it exists
UPDATE `company_settings` 
SET 
    `country` = 'Deutschland',
    `footer_text` = 'Vielen Dank für Ihr Vertrauen!\n\nBei Fragen zu dieser Rechnung stehen wir Ihnen gerne zur Verfügung.',
    `payment_terms_text` = 'Zahlbar innerhalb von 30 Tagen ohne Abzug.\n\nBei Zahlungsverzug behalten wir uns vor, Verzugszinsen in Höhe von 9 Prozentpunkten über dem Basiszinssatz zu berechnen.'
WHERE `id` = 1;

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS `idx_company_settings_iban` ON `company_settings` (`iban`);
CREATE INDEX IF NOT EXISTS `idx_company_settings_register_number` ON `company_settings` (`register_number`);

-- Show completion status
SELECT 'German business fields migration completed successfully!' as status;