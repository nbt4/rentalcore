-- ============================================================================
-- Invoice System Migration
-- Description: Creates comprehensive invoice system with customizable layouts
-- Date: 2025-06-12
-- ============================================================================

-- Company/Business Information Table
CREATE TABLE IF NOT EXISTS `company_settings` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `company_name` VARCHAR(200) NOT NULL,
    `address_line1` VARCHAR(200) NULL,
    `address_line2` VARCHAR(200) NULL,
    `city` VARCHAR(100) NULL,
    `state` VARCHAR(100) NULL,
    `postal_code` VARCHAR(20) NULL,
    `country` VARCHAR(100) NULL,
    `phone` VARCHAR(50) NULL,
    `email` VARCHAR(200) NULL,
    `website` VARCHAR(200) NULL,
    `tax_number` VARCHAR(100) NULL,
    `vat_number` VARCHAR(100) NULL,
    `logo_path` VARCHAR(500) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Invoice Templates Table
CREATE TABLE IF NOT EXISTS `invoice_templates` (
    `template_id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(100) NOT NULL,
    `description` TEXT NULL,
    `html_template` LONGTEXT NOT NULL,
    `css_styles` LONGTEXT NULL,
    `is_default` TINYINT(1) NOT NULL DEFAULT 0,
    `is_active` TINYINT(1) NOT NULL DEFAULT 1,
    `created_by` INT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX `idx_invoice_templates_default` (`is_default`),
    INDEX `idx_invoice_templates_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Invoices Table
CREATE TABLE IF NOT EXISTS `invoices` (
    `invoice_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `invoice_number` VARCHAR(50) NOT NULL UNIQUE,
    `customer_id` INT NOT NULL,
    `job_id` INT NULL,
    `template_id` INT NULL,
    `status` ENUM('draft', 'sent', 'paid', 'overdue', 'cancelled') NOT NULL DEFAULT 'draft',
    `issue_date` DATE NOT NULL,
    `due_date` DATE NOT NULL,
    `payment_terms` VARCHAR(100) NULL,
    
    -- Financial Details
    `subtotal` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `tax_rate` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    `tax_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `discount_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `total_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `paid_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `balance_due` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    
    -- Additional Information
    `notes` TEXT NULL,
    `terms_conditions` TEXT NULL,
    `internal_notes` TEXT NULL,
    
    -- Tracking
    `sent_at` TIMESTAMP NULL,
    `paid_at` TIMESTAMP NULL,
    `created_by` INT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Indexes
    INDEX `idx_invoices_customer` (`customer_id`),
    INDEX `idx_invoices_job` (`job_id`),
    INDEX `idx_invoices_status` (`status`),
    INDEX `idx_invoices_issue_date` (`issue_date`),
    INDEX `idx_invoices_due_date` (`due_date`),
    INDEX `idx_invoices_number` (`invoice_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Invoice Line Items Table
CREATE TABLE IF NOT EXISTS `invoice_line_items` (
    `line_item_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `invoice_id` BIGINT UNSIGNED NOT NULL,
    `item_type` ENUM('device', 'service', 'package', 'custom') NOT NULL DEFAULT 'custom',
    `device_id` VARCHAR(50) NULL,
    `package_id` INT NULL,
    `description` TEXT NOT NULL,
    `quantity` DECIMAL(10,2) NOT NULL DEFAULT 1.00,
    `unit_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `total_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    `rental_start_date` DATE NULL,
    `rental_end_date` DATE NULL,
    `rental_days` INT NULL,
    `sort_order` INT UNSIGNED NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign Keys
    FOREIGN KEY (`invoice_id`) REFERENCES `invoices`(`invoice_id`) ON DELETE CASCADE,
    
    -- Indexes
    INDEX `idx_invoice_line_items_invoice` (`invoice_id`),
    INDEX `idx_invoice_line_items_device` (`device_id`),
    INDEX `idx_invoice_line_items_package` (`package_id`),
    INDEX `idx_invoice_line_items_type` (`item_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Invoice Settings Table
CREATE TABLE IF NOT EXISTS `invoice_settings` (
    `setting_id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `setting_key` VARCHAR(100) NOT NULL UNIQUE,
    `setting_value` TEXT NULL,
    `setting_type` ENUM('text', 'number', 'boolean', 'json') NOT NULL DEFAULT 'text',
    `description` TEXT NULL,
    `updated_by` INT NULL,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX `idx_invoice_settings_key` (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Invoice Payments Table (for tracking partial payments)
CREATE TABLE IF NOT EXISTS `invoice_payments` (
    `payment_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `invoice_id` BIGINT UNSIGNED NOT NULL,
    `amount` DECIMAL(12,2) NOT NULL,
    `payment_method` VARCHAR(100) NULL,
    `payment_date` DATE NOT NULL,
    `reference_number` VARCHAR(100) NULL,
    `notes` TEXT NULL,
    `created_by` INT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (`invoice_id`) REFERENCES `invoices`(`invoice_id`) ON DELETE CASCADE,
    
    INDEX `idx_invoice_payments_invoice` (`invoice_id`),
    INDEX `idx_invoice_payments_date` (`payment_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Insert default company settings
INSERT IGNORE INTO `company_settings` (
    `company_name`, 
    `address_line1`, 
    `city`, 
    `country`, 
    `email`
) VALUES (
    'RentalCore Company', 
    '123 Business Street', 
    'Business City', 
    'Germany', 
    'info@rentalcore.com'
);

-- Insert default invoice settings
INSERT IGNORE INTO `invoice_settings` (`setting_key`, `setting_value`, `setting_type`, `description`) VALUES
('invoice_number_prefix', 'INV-', 'text', 'Prefix for invoice numbers'),
('invoice_number_format', '{prefix}{year}{month}{sequence:4}', 'text', 'Format for invoice numbers'),
('default_payment_terms', '30', 'number', 'Default payment terms in days'),
('default_tax_rate', '19.00', 'number', 'Default tax rate percentage'),
('auto_calculate_rental_days', 'true', 'boolean', 'Automatically calculate rental days'),
('show_logo_on_invoice', 'true', 'boolean', 'Show company logo on invoices'),
('currency_symbol', '€', 'text', 'Currency symbol to display'),
('currency_code', 'EUR', 'text', 'Currency code'),
('date_format', 'DD.MM.YYYY', 'text', 'Date format for invoices');

-- Insert default invoice template
INSERT IGNORE INTO `invoice_templates` (
    `name`, 
    `description`, 
    `html_template`, 
    `is_default`, 
    `is_active`
) VALUES (
    'Professional Template',
    'Clean professional invoice template with logo support',
    '<!-- Default template will be inserted via Go code -->',
    1,
    1
);

-- Create views for reporting
CREATE OR REPLACE VIEW `vw_invoice_summary` AS
SELECT 
    i.invoice_id,
    i.invoice_number,
    i.status,
    i.issue_date,
    i.due_date,
    i.total_amount,
    i.paid_amount,
    i.balance_due,
    c.customerID as customer_id,
    COALESCE(c.companyname, CONCAT(c.firstname, ' ', c.lastname)) as customer_name,
    j.jobID as job_id,
    j.description as job_description,
    DATEDIFF(CURDATE(), i.due_date) as days_overdue,
    COUNT(ili.line_item_id) as item_count
FROM invoices i
LEFT JOIN customers c ON i.customer_id = c.customerID
LEFT JOIN jobs j ON i.job_id = j.jobID
LEFT JOIN invoice_line_items ili ON i.invoice_id = ili.invoice_id
GROUP BY i.invoice_id, i.invoice_number, i.status, i.issue_date, i.due_date, 
         i.total_amount, i.paid_amount, i.balance_due, c.customerID, 
         c.companyname, c.firstname, c.lastname, j.jobID, j.description;

-- Show completion status
SELECT 'Invoice system migration completed successfully!' as status;
SELECT COUNT(*) as company_settings_count FROM company_settings;
SELECT COUNT(*) as invoice_templates_count FROM invoice_templates;
SELECT COUNT(*) as invoice_settings_count FROM invoice_settings;