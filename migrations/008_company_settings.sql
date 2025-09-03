-- Migration: Company Settings Table
-- Description: Creates company_settings table for storing company information
-- Required for German invoice compliance (GoBD)

CREATE TABLE IF NOT EXISTS `company_settings` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `company_name` VARCHAR(255) NOT NULL,
    `address_line1` VARCHAR(255) NULL,
    `address_line2` VARCHAR(255) NULL,
    `city` VARCHAR(100) NULL,
    `state` VARCHAR(100) NULL,
    `postal_code` VARCHAR(20) NULL,
    `country` VARCHAR(100) NULL DEFAULT 'Deutschland',
    `phone` VARCHAR(50) NULL,
    `email` VARCHAR(255) NULL,
    `website` VARCHAR(255) NULL,
    `tax_number` VARCHAR(50) NULL COMMENT 'Steuernummer für deutsche Unternehmen',
    `vat_number` VARCHAR(50) NULL COMMENT 'USt-IdNr für EU-Unternehmen',
    `logo_path` VARCHAR(500) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Company settings for invoice generation and German compliance';

-- Insert default company settings if none exist
INSERT IGNORE INTO `company_settings` (
    `id`,
    `company_name`,
    `address_line1`,
    `city`,
    `postal_code`,
    `country`,
    `phone`,
    `email`,
    `tax_number`,
    `vat_number`
) VALUES (
    1,
    'TS RentalCore GmbH',
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
);

-- Add index for faster lookups (should only be one record anyway)
CREATE INDEX IF NOT EXISTS `idx_company_settings_updated` ON `company_settings` (`updated_at`);

-- Ensure only one company settings record exists (business rule)
-- This will be enforced in the application layer