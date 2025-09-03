-- ============================================================================
-- RentalCore Database Setup Script
-- ============================================================================
-- Professional Equipment Rental Management System
-- GitHub: https://github.com/nbt4/RentalCore
-- Docker Hub: https://hub.docker.com/r/nbt4/rentalcore
-- 
-- This script creates the complete database schema and sample data
-- for RentalCore deployment.
-- ============================================================================

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

-- ============================================================================
-- Create Database (uncomment if creating new database)
-- ============================================================================
-- CREATE DATABASE IF NOT EXISTS `rentalcore` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
-- USE `rentalcore`;

-- ============================================================================
-- Analytics Cache Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS `analytics_cache` (
  `cacheID` int NOT NULL AUTO_INCREMENT,
  `metric_name` varchar(100) NOT NULL,
  `period_type` enum('daily','weekly','monthly','yearly') NOT NULL,
  `period_date` date NOT NULL,
  `value` decimal(15,4) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`cacheID`),
  UNIQUE KEY `unique_metric_period` (`metric_name`, `period_type`, `period_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Categories Table - Equipment Categories
-- ============================================================================
CREATE TABLE IF NOT EXISTS `categories` (
  `categoryID` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `description` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`categoryID`),
  UNIQUE KEY `unique_category_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Products Table - Equipment Types
-- ============================================================================
CREATE TABLE IF NOT EXISTS `products` (
  `productID` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `description` text,
  `categoryID` int NOT NULL,
  `itemcostperday` decimal(10,2) NOT NULL DEFAULT '0.00',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`productID`),
  KEY `idx_category` (`categoryID`),
  CONSTRAINT `fk_products_category` FOREIGN KEY (`categoryID`) REFERENCES `categories` (`categoryID`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Customers Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS `customers` (
  `customerID` int NOT NULL AUTO_INCREMENT,
  `firstname` varchar(50) DEFAULT NULL,
  `lastname` varchar(50) DEFAULT NULL,
  `companyname` varchar(100) DEFAULT NULL,
  `email` varchar(100) NOT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `address` text,
  `city` varchar(50) DEFAULT NULL,
  `postal_code` varchar(20) DEFAULT NULL,
  `country` varchar(50) DEFAULT 'Germany',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`customerID`),
  UNIQUE KEY `unique_email` (`email`),
  KEY `idx_customer_search` (`lastname`, `companyname`, `email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Job Statuses Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS `statuses` (
  `statusID` int NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `color` varchar(7) DEFAULT '#6B7280',
  `is_active` tinyint(1) DEFAULT '1',
  PRIMARY KEY (`statusID`),
  UNIQUE KEY `unique_status_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Jobs Table - Rental Jobs
-- ============================================================================
CREATE TABLE IF NOT EXISTS `jobs` (
  `jobID` int NOT NULL AUTO_INCREMENT,
  `customerID` int NOT NULL,
  `statusID` int NOT NULL DEFAULT '1',
  `description` text,
  `startDate` datetime NOT NULL,
  `endDate` datetime DEFAULT NULL,
  `location` varchar(255) DEFAULT NULL,
  `revenue` decimal(10,2) DEFAULT '0.00',
  `final_revenue` decimal(10,2) DEFAULT NULL,
  `discount` decimal(5,2) DEFAULT '0.00',
  `discount_type` enum('percent','amount') DEFAULT 'percent',
  `notes` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`jobID`),
  KEY `idx_customer` (`customerID`),
  KEY `idx_status` (`statusID`),
  KEY `idx_dates` (`startDate`, `endDate`),
  CONSTRAINT `fk_jobs_customer` FOREIGN KEY (`customerID`) REFERENCES `customers` (`customerID`) ON DELETE RESTRICT,
  CONSTRAINT `fk_jobs_status` FOREIGN KEY (`statusID`) REFERENCES `statuses` (`statusID`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Devices Table - Equipment Inventory
-- ============================================================================
CREATE TABLE IF NOT EXISTS `devices` (
  `deviceID` varchar(50) NOT NULL,
  `productID` int NOT NULL,
  `serialnumber` varchar(100) DEFAULT NULL,
  `status` enum('available','checked out','maintenance','retired') NOT NULL DEFAULT 'available',
  `condition_notes` text,
  `purchase_date` date DEFAULT NULL,
  `purchase_price` decimal(10,2) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`deviceID`),
  KEY `idx_product` (`productID`),
  KEY `idx_status` (`status`),
  KEY `idx_serial` (`serialnumber`),
  CONSTRAINT `fk_devices_product` FOREIGN KEY (`productID`) REFERENCES `products` (`productID`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Job Devices Table - Many-to-Many Relationship
-- ============================================================================
CREATE TABLE IF NOT EXISTS `jobdevices` (
  `id` int NOT NULL AUTO_INCREMENT,
  `jobID` int NOT NULL,
  `deviceID` varchar(50) NOT NULL,
  `custom_price` decimal(10,2) DEFAULT NULL,
  `assigned_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `returned_at` timestamp NULL DEFAULT NULL,
  `condition_out` text,
  `condition_in` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_job_device` (`jobID`, `deviceID`),
  KEY `idx_device` (`deviceID`),
  CONSTRAINT `fk_jobdevices_job` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE CASCADE,
  CONSTRAINT `fk_jobdevices_device` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Users Table - System Users
-- ============================================================================
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL,
  `email` varchar(100) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `firstname` varchar(50) DEFAULT NULL,
  `lastname` varchar(50) DEFAULT NULL,
  `role` enum('admin','manager','user') NOT NULL DEFAULT 'user',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `last_login` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_username` (`username`),
  UNIQUE KEY `unique_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Insert Sample Data
-- ============================================================================

-- Sample Categories
INSERT INTO `categories` (`name`, `description`) VALUES
('Audio Equipment', 'Professional audio equipment including speakers, microphones, and mixing consoles'),
('Lighting Equipment', 'Stage and event lighting equipment'),
('Video Equipment', 'Cameras, projectors, and video production equipment'),
('Power & Cables', 'Power distribution, cables, and connectivity equipment'),
('Staging & Rigging', 'Stage platforms, trusses, and rigging equipment');

-- Sample Products
INSERT INTO `products` (`name`, `description`, `categoryID`, `itemcostperday`) VALUES
('Professional Speaker System', '2-way powered speaker system with 15" woofer', 1, 45.00),
('Wireless Microphone Set', 'Professional UHF wireless microphone system', 1, 25.00),
('LED Par Light', 'RGBA LED Par light with DMX control', 2, 15.00),
('Moving Head Light', 'Professional moving head spot light with gobos', 2, 35.00),
('HD Camera', 'Professional HD video camera with tripod', 3, 75.00),
('4K Projector', 'High-brightness 4K projector for large venues', 3, 120.00),
('Power Distribution Unit', '32A 3-phase power distribution with CEE outputs', 4, 20.00),
('XLR Cable 10m', 'Professional XLR microphone cable', 4, 3.00);

-- Sample Statuses
INSERT INTO `statuses` (`name`, `description`, `color`) VALUES
('Planning', 'Job is being planned and prepared', '#3B82F6'),
('Active', 'Equipment is currently deployed for this job', '#10B981'),
('Completed', 'Job has been completed successfully', '#22C55E'),
('Cancelled', 'Job has been cancelled', '#EF4444'),
('On Hold', 'Job is temporarily on hold', '#F59E0B');

-- Sample Customers
INSERT INTO `customers` (`firstname`, `lastname`, `companyname`, `email`, `phone`, `address`, `city`, `postal_code`) VALUES
('John', 'Smith', 'Smith Events Ltd', 'john@smithevents.com', '+49 30 12345678', 'Alexanderplatz 1', 'Berlin', '10178'),
('Maria', 'Rodriguez', 'Rodriguez Productions', 'maria@rodriguezprod.com', '+49 89 87654321', 'Marienplatz 5', 'Munich', '80331'),
('David', 'Johnson', NULL, 'david.johnson@email.com', '+49 40 55566677', 'Reeperbahn 20', 'Hamburg', '20359'),
('Sarah', 'Wilson', 'Wilson Creative Agency', 'sarah@wilsoncreative.de', '+49 221 99887766', 'Dom Platz 3', 'Cologne', '50667'),
('Michael', 'Brown', 'Brown Entertainment', 'mike@brownent.com', '+49 711 44433322', 'Königstraße 15', 'Stuttgart', '70173');

-- Sample Devices
INSERT INTO `devices` (`deviceID`, `productID`, `serialnumber`, `status`, `purchase_date`, `purchase_price`) VALUES
('SPK001', 1, 'QSC-KW153-001', 'available', '2023-01-15', 1200.00),
('SPK002', 1, 'QSC-KW153-002', 'available', '2023-01-15', 1200.00),
('MIC001', 2, 'SHURE-ULXD24-001', 'available', '2023-02-20', 800.00),
('MIC002', 2, 'SHURE-ULXD24-002', 'available', '2023-02-20', 800.00),
('LED001', 3, 'CHAUVET-COLORado-001', 'available', '2023-03-10', 300.00),
('LED002', 3, 'CHAUVET-COLORado-002', 'available', '2023-03-10', 300.00),
('MOV001', 4, 'MARTIN-MAC-VIPER-001', 'available', '2023-04-05', 2500.00),
('CAM001', 5, 'SONY-PXW-Z190-001', 'available', '2023-05-12', 3500.00),
('PRJ001', 6, 'EPSON-EB-PU2010B-001', 'available', '2023-06-18', 8500.00),
('PWR001', 7, 'DISTRO-32A-001', 'available', '2023-07-25', 450.00);

-- Sample Jobs (recent completed jobs for analytics)
INSERT INTO `jobs` (`customerID`, `statusID`, `description`, `startDate`, `endDate`, `location`, `revenue`, `final_revenue`) VALUES
(1, 3, 'Corporate Conference - Berlin Tech Summit', '2024-01-15 08:00:00', '2024-01-17 18:00:00', 'Berlin Convention Center', 1200.00, 1150.00),
(2, 3, 'Wedding Reception - Rodriguez Family', '2024-01-20 16:00:00', '2024-01-21 02:00:00', 'Munich Marriott Hotel', 800.00, 850.00),
(3, 3, 'Birthday Party Setup', '2024-02-03 14:00:00', '2024-02-04 01:00:00', 'Private Residence Hamburg', 450.00, 450.00),
(4, 2, 'Product Launch Event', '2024-02-15 09:00:00', '2024-02-15 21:00:00', 'Cologne Trade Center', 2100.00, NULL),
(5, 1, 'Annual Company Meeting', '2024-03-01 08:00:00', '2024-03-01 17:00:00', 'Stuttgart Conference Hall', 900.00, NULL);

-- Sample Job Device Assignments
INSERT INTO `jobdevices` (`jobID`, `deviceID`, `custom_price`) VALUES
-- Job 1 devices
(1, 'SPK001', NULL),
(1, 'SPK002', NULL),
(1, 'MIC001', NULL),
(1, 'MIC002', NULL),
(1, 'LED001', NULL),
(1, 'LED002', NULL),
(1, 'PWR001', NULL),
-- Job 2 devices  
(2, 'SPK001', NULL),
(2, 'MIC001', NULL),
(2, 'LED001', 12.00),
(2, 'LED002', 12.00),
-- Job 3 devices
(3, 'MIC001', NULL),
(3, 'LED001', NULL);

-- Sample Default Admin User (password: admin123)
-- Note: This is a demo password, change immediately in production!
INSERT INTO `users` (`username`, `email`, `password_hash`, `firstname`, `lastname`, `role`) VALUES
('admin', 'admin@rentalcore.local', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LdTcTquUuP5wNqb9C', 'Admin', 'User', 'admin');

-- ============================================================================
-- Create Indexes for Performance
-- ============================================================================
CREATE INDEX `idx_jobs_revenue_period` ON `jobs` (`endDate`, `final_revenue`, `revenue`);
CREATE INDEX `idx_analytics_cache_lookup` ON `analytics_cache` (`metric_name`, `period_type`, `updated_at`);
CREATE INDEX `idx_devices_product_status` ON `devices` (`productID`, `status`);
CREATE INDEX `idx_jobdevices_dates` ON `jobdevices` (`assigned_at`, `returned_at`);

-- ============================================================================
-- Set Auto Increment Starting Values
-- ============================================================================
ALTER TABLE `categories` AUTO_INCREMENT = 100;
ALTER TABLE `products` AUTO_INCREMENT = 1000;
ALTER TABLE `customers` AUTO_INCREMENT = 10000;
ALTER TABLE `statuses` AUTO_INCREMENT = 10;
ALTER TABLE `jobs` AUTO_INCREMENT = 100000;
ALTER TABLE `users` AUTO_INCREMENT = 1;
ALTER TABLE `analytics_cache` AUTO_INCREMENT = 1;
ALTER TABLE `jobdevices` AUTO_INCREMENT = 1;

COMMIT;

-- ============================================================================
-- Setup Complete!
-- ============================================================================
-- Database schema and sample data created successfully.
-- 
-- Default Admin Login:
-- Username: admin
-- Password: admin123 (CHANGE IMMEDIATELY!)
-- 
-- Next Steps:
-- 1. Configure your .env file with database credentials
-- 2. Start the RentalCore application: docker-compose up -d
-- 3. Access the application at http://localhost:8080
-- 4. Login with admin/admin123 and change the password
-- 5. Add your own equipment, customers, and start managing rentals!
-- 
-- For deployment help: https://github.com/nbt4/RentalCore
-- ============================================================================

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;