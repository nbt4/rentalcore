-- Migration 024: Add pack workflow functionality
-- Adds pack status tracking to jobs and device relationships

-- Add pack workflow columns to jobdevices table
ALTER TABLE `jobdevices`
ADD COLUMN `pack_status` ENUM('pending','packed','issued','returned') DEFAULT 'pending' NOT NULL AFTER `custom_price`,
ADD COLUMN `pack_ts` DATETIME NULL AFTER `pack_status`,
ADD INDEX `idx_jobdevices_pack_status` (`pack_status`),
ADD INDEX `idx_jobdevices_job_pack` (`jobID`, `pack_status`);

-- Create job_device_events audit table
CREATE TABLE `job_device_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `jobID` INT NOT NULL,
  `deviceID` VARCHAR(50) NOT NULL,
  `event_type` ENUM('scanned','packed','issued','returned','unpacked') NOT NULL,
  `actor` VARCHAR(100) DEFAULT NULL COMMENT 'User who performed the action',
  `timestamp` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `metadata` JSON DEFAULT NULL COMMENT 'Additional event data',
  PRIMARY KEY (`id`),
  KEY `idx_job_device_events_job` (`jobID`),
  KEY `idx_job_device_events_device` (`deviceID`),
  KEY `idx_job_device_events_type` (`event_type`),
  KEY `idx_job_device_events_timestamp` (`timestamp`),
  CONSTRAINT `fk_job_device_events_job` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE CASCADE,
  CONSTRAINT `fk_job_device_events_device` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Create product_images table
CREATE TABLE `product_images` (
  `imageID` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `productID` INT NOT NULL,
  `filename` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(255) DEFAULT NULL,
  `file_path` VARCHAR(500) NOT NULL,
  `file_size` BIGINT UNSIGNED DEFAULT NULL,
  `mime_type` VARCHAR(100) DEFAULT NULL,
  `is_primary` BOOLEAN DEFAULT FALSE,
  `alt_text` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`imageID`),
  KEY `idx_product_images_product` (`productID`),
  KEY `idx_product_images_primary` (`productID`, `is_primary`),
  CONSTRAINT `fk_product_images_product` FOREIGN KEY (`productID`) REFERENCES `products` (`productID`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Create view for job pack progress
CREATE VIEW `v_job_pack_progress` AS
SELECT
    j.jobID,
    j.description as job_description,
    COUNT(jd.deviceID) as total_devices,
    COUNT(CASE WHEN jd.pack_status = 'packed' THEN 1 END) as packed_devices,
    COUNT(CASE WHEN jd.pack_status = 'issued' THEN 1 END) as issued_devices,
    COUNT(CASE WHEN jd.pack_status = 'returned' THEN 1 END) as returned_devices,
    COUNT(CASE WHEN jd.pack_status = 'pending' THEN 1 END) as pending_devices,
    CASE
        WHEN COUNT(jd.deviceID) = 0 THEN 100.0
        ELSE ROUND((COUNT(CASE WHEN jd.pack_status = 'packed' THEN 1 END) * 100.0) / COUNT(jd.deviceID), 2)
    END as pack_progress_percent
FROM jobs j
LEFT JOIN jobdevices jd ON j.jobID = jd.jobID
GROUP BY j.jobID, j.description;

-- Set all existing jobdevices to 'pending' status (they are already there by default)
-- This migration is backward compatible - no data loss