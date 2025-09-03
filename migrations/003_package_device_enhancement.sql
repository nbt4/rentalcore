-- ============================================================================
-- Migration: Package-Device Relationship Tables
-- Description: Creates junction table and related structures for connecting 
--              equipment packages with devices
-- Date: 2025-06-12
-- ============================================================================

-- Create the package_devices junction table
CREATE TABLE IF NOT EXISTS `package_devices` (
    `packageID` INT UNSIGNED NOT NULL,
    `deviceID` VARCHAR(255) NOT NULL,
    `quantity` INT UNSIGNED NOT NULL DEFAULT 1,
    `custom_price` DECIMAL(12,2) NULL,
    `is_required` BOOLEAN NOT NULL DEFAULT FALSE,
    `notes` TEXT NULL,
    `sort_order` INT UNSIGNED NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Composite primary key
    PRIMARY KEY (`packageID`, `deviceID`),
    
    -- Foreign key constraints
    CONSTRAINT `fk_package_devices_package`
        FOREIGN KEY (`packageID`) 
        REFERENCES `equipment_packages`(`packageID`)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
        
    CONSTRAINT `fk_package_devices_device`
        FOREIGN KEY (`deviceID`) 
        REFERENCES `devices`(`deviceID`)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    
    -- Indexes for performance
    INDEX `idx_package_devices_package` (`packageID`),
    INDEX `idx_package_devices_device` (`deviceID`),
    INDEX `idx_package_devices_required` (`is_required`),
    INDEX `idx_package_devices_sort` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Optional: Package Categories Table (for organizing packages)
-- ============================================================================

CREATE TABLE IF NOT EXISTS `package_categories` (
    `categoryID` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(100) NOT NULL,
    `description` TEXT NULL,
    `color` VARCHAR(7) NULL COMMENT 'Hex color code for UI',
    `sort_order` INT UNSIGNED NULL,
    `is_active` BOOLEAN NOT NULL DEFAULT TRUE,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Unique constraint
    UNIQUE KEY `uk_package_categories_name` (`name`),
    
    -- Indexes
    INDEX `idx_package_categories_active` (`is_active`),
    INDEX `idx_package_categories_sort` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add category relationship to equipment_packages table
ALTER TABLE `equipment_packages` 
ADD COLUMN `categoryID` INT UNSIGNED NULL AFTER `description`,
ADD CONSTRAINT `fk_equipment_packages_category`
    FOREIGN KEY (`categoryID`) 
    REFERENCES `package_categories`(`categoryID`)
    ON DELETE SET NULL
    ON UPDATE CASCADE,
ADD INDEX `idx_equipment_packages_category` (`categoryID`);

-- ============================================================================
-- Package Usage Tracking (optional - for analytics)
-- ============================================================================

CREATE TABLE IF NOT EXISTS `package_usage` (
    `usageID` INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `packageID` INT UNSIGNED NOT NULL,
    `jobID` INT UNSIGNED NULL,
    `customerID` INT UNSIGNED NULL,
    `used_date` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `rental_days` INT UNSIGNED NULL,
    `total_price` DECIMAL(12,2) NULL,
    `notes` TEXT NULL,
    
    -- Foreign key constraints
    CONSTRAINT `fk_package_usage_package`
        FOREIGN KEY (`packageID`) 
        REFERENCES `equipment_packages`(`packageID`)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
        
    -- Note: jobs and customers foreign keys would need to reference existing tables
    -- Uncomment and adjust if those tables exist:
    -- CONSTRAINT `fk_package_usage_job`
    --     FOREIGN KEY (`jobID`) 
    --     REFERENCES `jobs`(`jobID`)
    --     ON DELETE SET NULL
    --     ON UPDATE CASCADE,
    --     
    -- CONSTRAINT `fk_package_usage_customer`
    --     FOREIGN KEY (`customerID`) 
    --     REFERENCES `customers`(`customerID`)
    --     ON DELETE SET NULL
    --     ON UPDATE CASCADE,
    
    -- Indexes
    INDEX `idx_package_usage_package` (`packageID`),
    INDEX `idx_package_usage_job` (`jobID`),
    INDEX `idx_package_usage_customer` (`customerID`),
    INDEX `idx_package_usage_date` (`used_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Insert default package categories (optional)
-- ============================================================================

INSERT IGNORE INTO `package_categories` (`name`, `description`, `color`, `sort_order`) VALUES
('Basic', 'Basic equipment packages for standard events', '#007bff', 1),
('Premium', 'Premium packages with high-end equipment', '#28a745', 2),
('Specialized', 'Specialized packages for specific event types', '#ffc107', 3),
('Seasonal', 'Seasonal packages for holidays and special occasions', '#17a2b8', 4),
('Custom', 'Custom packages created for specific customers', '#6c757d', 5);

-- ============================================================================
-- Update equipment_packages table usage counter (trigger)
-- ============================================================================

DELIMITER $$

CREATE TRIGGER `tr_package_usage_increment` 
AFTER INSERT ON `package_usage`
FOR EACH ROW
BEGIN
    UPDATE `equipment_packages` 
    SET `usageCount` = `usageCount` + 1 
    WHERE `packageID` = NEW.packageID;
END$$

DELIMITER ;

-- ============================================================================
-- Useful Views for querying package-device relationships
-- ============================================================================

-- View: Package details with device count and total value
CREATE OR REPLACE VIEW `vw_package_summary` AS
SELECT 
    ep.packageID,
    ep.name as packageName,
    ep.description,
    ep.packagePrice,
    ep.discountPercent,
    ep.minRentalDays,
    ep.isActive,
    ep.usageCount,
    pc.name as categoryName,
    COUNT(pd.deviceID) as deviceCount,
    SUM(pd.quantity) as totalDevices,
    COUNT(CASE WHEN pd.is_required = 1 THEN 1 END) as requiredDevices,
    COUNT(CASE WHEN pd.is_required = 0 THEN 1 END) as optionalDevices,
    ep.createdAt,
    ep.updatedAt
FROM equipment_packages ep
LEFT JOIN package_categories pc ON ep.categoryID = pc.categoryID
LEFT JOIN package_devices pd ON ep.packageID = pd.packageID
GROUP BY ep.packageID, ep.name, ep.description, ep.packagePrice, 
         ep.discountPercent, ep.minRentalDays, ep.isActive, 
         ep.usageCount, pc.name, ep.createdAt, ep.updatedAt;

-- View: Package devices with product details
CREATE OR REPLACE VIEW `vw_package_devices_detail` AS
SELECT 
    pd.packageID,
    ep.name as packageName,
    pd.deviceID,
    d.serialNumber,
    d.status as deviceStatus,
    p.name as productName,
    p.category as productCategory,
    p.subcategory as productSubcategory,
    p.itemCostPerDay as defaultPrice,
    pd.custom_price,
    COALESCE(pd.custom_price, p.itemCostPerDay) as effectivePrice,
    pd.quantity,
    pd.is_required,
    pd.notes,
    pd.sort_order,
    (COALESCE(pd.custom_price, p.itemCostPerDay) * pd.quantity) as lineTotal
FROM package_devices pd
INNER JOIN equipment_packages ep ON pd.packageID = ep.packageID
INNER JOIN devices d ON pd.deviceID = d.deviceID
LEFT JOIN products p ON d.productID = p.productID
ORDER BY pd.packageID, pd.sort_order, pd.deviceID;