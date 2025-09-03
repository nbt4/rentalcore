-- phpMyAdmin SQL Dump
-- version 5.2.2
-- https://www.phpmyadmin.net/
--
-- Host: mysql
-- Generation Time: Jul 23, 2025 at 09:47 PM
-- Server version: 9.2.0
-- PHP Version: 8.2.27

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `TS-Lager`
--

-- --------------------------------------------------------

--
-- Table structure for table `analytics_cache`
--

CREATE TABLE `analytics_cache` (
  `cacheID` int NOT NULL,
  `metric_name` varchar(100) NOT NULL,
  `period_type` enum('daily','weekly','monthly','yearly') NOT NULL,
  `period_date` date NOT NULL,
  `value` decimal(15,4) DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `archived_documents`
--

CREATE TABLE `archived_documents` (
  `id` bigint UNSIGNED NOT NULL,
  `document_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `document_id` bigint UNSIGNED NOT NULL,
  `original_hash` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `archived_data` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `archived_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `retention_until` timestamp NOT NULL,
  `legal_basis` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `archive_format` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'json',
  `compression_used` tinyint(1) DEFAULT '0',
  `encryption_used` tinyint(1) DEFAULT '0',
  `archive_path` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `audit_events`
--

CREATE TABLE `audit_events` (
  `id` bigint UNSIGNED NOT NULL,
  `event_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `action` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `changes` json DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `timestamp` datetime(3) NOT NULL,
  `ip_address` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) DEFAULT NULL,
  `object_type` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `object_id` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `username` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `old_values` text COLLATE utf8mb4_unicode_ci,
  `new_values` text COLLATE utf8mb4_unicode_ci,
  `session_id` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `context` json DEFAULT NULL,
  `event_hash` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `previous_hash` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_compliant` tinyint(1) DEFAULT '1',
  `retention_date` datetime(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `audit_log`
--

CREATE TABLE `audit_log` (
  `auditID` bigint NOT NULL,
  `userID` bigint UNSIGNED DEFAULT NULL,
  `action` varchar(100) NOT NULL,
  `entity_type` varchar(50) NOT NULL,
  `entity_id` varchar(50) NOT NULL,
  `old_values` json DEFAULT NULL,
  `new_values` json DEFAULT NULL,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` text,
  `session_id` varchar(191) DEFAULT NULL,
  `timestamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `audit_logs`
--

CREATE TABLE `audit_logs` (
  `id` bigint UNSIGNED NOT NULL,
  `entity_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `entity_id` bigint UNSIGNED NOT NULL,
  `action` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` bigint UNSIGNED DEFAULT NULL,
  `changes` json DEFAULT NULL,
  `metadata` json DEFAULT NULL,
  `hash` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `previous_hash` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `brands`
--

CREATE TABLE `brands` (
  `brandID` int NOT NULL,
  `name` varchar(100) NOT NULL,
  `manufacturerID` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `cables`
--

CREATE TABLE `cables` (
  `cableID` int NOT NULL,
  `connector1` int NOT NULL,
  `connector2` int NOT NULL,
  `typ` int NOT NULL,
  `length` decimal(10,2) NOT NULL COMMENT 'in metern',
  `mm2` decimal(10,2) NOT NULL COMMENT 'Kabelquerschnitt in mm^2',
  `name` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `cables`
--
DELIMITER $$
CREATE TRIGGER `cables_before_insert` BEFORE INSERT ON `cables` FOR EACH ROW BEGIN
  DECLARE typ_name VARCHAR(50);
  DECLARE conn1_name VARCHAR(50);
  DECLARE conn2_name VARCHAR(50);

  -- Hole Namen des Typs
  SELECT name INTO typ_name
  FROM cable_types
  WHERE cable_typesID = NEW.typ;

  -- Hole Namen oder Abkürzung von Connector1
  SELECT IFNULL(abbreviation, name) INTO conn1_name
  FROM cable_connectors
  WHERE cable_connectorsID = NEW.connector1;

  -- Hole Namen oder Abkürzung von Connector2
  SELECT IFNULL(abbreviation, name) INTO conn2_name
  FROM cable_connectors
  WHERE cable_connectorsID = NEW.connector2;

  -- Setze den zusammengesetzten Namen
  SET NEW.name = CONCAT(typ_name, ' Kabel',' (', conn1_name, '-', conn2_name, ')', ' - ', ROUND(NEW.length, 2), ' m');
END
$$
DELIMITER ;
DELIMITER $$
CREATE TRIGGER `cables_before_update` BEFORE UPDATE ON `cables` FOR EACH ROW BEGIN
  DECLARE typ_name VARCHAR(50);
  DECLARE conn1_name VARCHAR(50);
  DECLARE conn2_name VARCHAR(50);

  -- Hole Namen des Typs
  SELECT name INTO typ_name
  FROM cable_types
  WHERE cable_typesID = NEW.typ;

  -- Hole Namen oder Abkürzung von Connector1
  SELECT IFNULL(abbreviation, name) INTO conn1_name
  FROM cable_connectors
  WHERE cable_connectorsID = NEW.connector1;

  -- Hole Namen oder Abkürzung von Connector2
  SELECT IFNULL(abbreviation, name) INTO conn2_name
  FROM cable_connectors
  WHERE cable_connectorsID = NEW.connector2;

  -- Setze den zusammengesetzten Namen
  SET NEW.name = CONCAT(typ_name, ' Kabel',' (', conn1_name, '-', conn2_name, ')', ' - ', ROUND(NEW.length, 2), ' m');
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Table structure for table `cable_connectors`
--

CREATE TABLE `cable_connectors` (
  `cable_connectorsID` int NOT NULL,
  `name` varchar(30) NOT NULL,
  `abbreviation` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `gender` enum('male','female') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `cable_types`
--

CREATE TABLE `cable_types` (
  `cable_typesID` int NOT NULL,
  `name` varchar(30) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `cases`
--

CREATE TABLE `cases` (
  `caseID` int NOT NULL,
  `name` varchar(30) NOT NULL,
  `description` text,
  `width` decimal(10,2) DEFAULT NULL,
  `height` decimal(10,2) DEFAULT NULL,
  `depth` decimal(10,2) DEFAULT NULL,
  `weight` decimal(10,2) DEFAULT NULL,
  `status` enum('free','rented','maintance','') NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `categories`
--

CREATE TABLE `categories` (
  `categoryID` int NOT NULL,
  `name` varchar(20) NOT NULL,
  `abbreviation` varchar(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `company_settings`
--

CREATE TABLE `company_settings` (
  `id` int NOT NULL,
  `company_name` longtext NOT NULL,
  `address_line1` longtext,
  `address_line2` longtext,
  `city` longtext,
  `state` longtext,
  `postal_code` longtext,
  `country` longtext,
  `phone` longtext,
  `email` longtext,
  `website` longtext,
  `tax_number` longtext,
  `vat_number` longtext,
  `logo_path` longtext,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `bank_name` longtext,
  `iban` longtext,
  `bic` longtext,
  `account_holder` longtext,
  `ceo_name` longtext,
  `register_court` longtext,
  `register_number` longtext,
  `footer_text` text,
  `payment_terms_text` text,
  `smtp_host` varchar(255) DEFAULT NULL,
  `smtp_port` int DEFAULT NULL,
  `smtp_username` varchar(255) DEFAULT NULL,
  `smtp_password` varchar(255) DEFAULT NULL,
  `smtp_from_email` varchar(255) DEFAULT NULL,
  `smtp_from_name` varchar(255) DEFAULT NULL,
  `smtp_use_tls` tinyint(1) DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `consent_records`
--

CREATE TABLE `consent_records` (
  `id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `data_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `purpose` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `consent_given` tinyint(1) NOT NULL,
  `consent_date` timestamp NOT NULL,
  `expiry_date` timestamp NULL DEFAULT NULL,
  `legal_basis` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `withdrawn_at` timestamp NULL DEFAULT NULL,
  `version` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ip_address` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `customers`
--

CREATE TABLE `customers` (
  `customerID` int NOT NULL,
  `companyname` varchar(100) DEFAULT NULL,
  `lastname` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `firstname` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `street` varchar(100) DEFAULT NULL,
  `housenumber` varchar(20) DEFAULT NULL,
  `ZIP` varchar(20) DEFAULT NULL,
  `city` varchar(50) DEFAULT NULL,
  `federalstate` varchar(50) DEFAULT NULL,
  `country` varchar(50) DEFAULT NULL,
  `phonenumber` varchar(20) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL,
  `customertype` varchar(50) DEFAULT NULL,
  `notes` text,
  `tax_number` varchar(50) DEFAULT NULL,
  `credit_limit` decimal(12,2) DEFAULT '0.00',
  `payment_terms` int DEFAULT '30',
  `preferred_payment_method` varchar(50) DEFAULT NULL,
  `customer_since` date DEFAULT NULL,
  `total_lifetime_value` decimal(12,2) DEFAULT '0.00',
  `last_job_date` date DEFAULT NULL,
  `rating` decimal(3,1) DEFAULT '5.0',
  `billing_address` text,
  `shipping_address` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `data_processing_records`
--

CREATE TABLE `data_processing_records` (
  `id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `data_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `processing_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `purpose` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `legal_basis` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data_controller` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data_processor` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `recipients` json DEFAULT NULL,
  `transfer_country` varchar(2) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `retention_period` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `processed_at` timestamp NOT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `data_subject_requests`
--

CREATE TABLE `data_subject_requests` (
  `id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `request_type` enum('access','rectification','erasure','portability','restriction','objection') COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('pending','processing','completed','rejected') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `description` text COLLATE utf8mb4_unicode_ci,
  `requested_at` timestamp NOT NULL,
  `processed_at` timestamp NULL DEFAULT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  `processor_id` bigint UNSIGNED DEFAULT NULL,
  `response` text COLLATE utf8mb4_unicode_ci,
  `response_data` longtext COLLATE utf8mb4_unicode_ci,
  `verification` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `devices`
--

CREATE TABLE `devices` (
  `deviceID` varchar(50) NOT NULL,
  `productID` int DEFAULT NULL,
  `serialnumber` varchar(50) DEFAULT NULL,
  `purchaseDate` date DEFAULT NULL,
  `lastmaintenance` date DEFAULT NULL,
  `nextmaintenance` date DEFAULT NULL,
  `insurancenumber` varchar(50) DEFAULT NULL,
  `status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT 'free',
  `insuranceID` int DEFAULT NULL,
  `qr_code` varchar(255) DEFAULT NULL,
  `current_location` varchar(100) DEFAULT NULL,
  `gps_latitude` decimal(10,8) DEFAULT NULL,
  `gps_longitude` decimal(11,8) DEFAULT NULL,
  `condition_rating` decimal(3,1) DEFAULT '5.0',
  `usage_hours` decimal(10,2) DEFAULT '0.00',
  `total_revenue` decimal(12,2) DEFAULT '0.00',
  `last_maintenance_cost` decimal(10,2) DEFAULT NULL,
  `notes` text,
  `barcode` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `devices`
--
DELIMITER $$
CREATE TRIGGER `devices` BEFORE INSERT ON `devices` FOR EACH ROW BEGIN
  DECLARE abkuerzung   VARCHAR(50);
  DECLARE pos_cat       INT;
  DECLARE next_counter  INT;

  -- 1) Abkürzung holen
  SELECT s.abbreviation
    INTO abkuerzung
    FROM subcategories s
    JOIN products      p ON s.subcategoryID = p.subcategoryID
   WHERE p.productID   = NEW.productID
   LIMIT 1;

  -- 2) pos_in_category holen
  SELECT p.pos_in_category
    INTO pos_cat
    FROM products p
   WHERE p.productID = NEW.productID;

  -- 3) Laufindex ermitteln (max. der letzten 3 Ziffern + 1)
  SELECT COALESCE(MAX(CAST(RIGHT(d.deviceID, 3) AS UNSIGNED)), 0) + 1
    INTO next_counter
    FROM devices d
   WHERE d.deviceID LIKE CONCAT(abkuerzung, pos_cat, '%');

  -- 4) deviceID zusammenbauen (ohne Bindestrich!)
  SET NEW.deviceID = CONCAT(
                        abkuerzung,
                        pos_cat,
                        LPAD(next_counter, 3, '0')
                      );
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Table structure for table `devicescases`
--

CREATE TABLE `devicescases` (
  `caseID` int NOT NULL,
  `deviceID` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `devicestatushistory`
--

CREATE TABLE `devicestatushistory` (
  `statushistoryID` int NOT NULL,
  `deviceID` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `date` datetime DEFAULT NULL,
  `status` varchar(50) DEFAULT NULL,
  `notes` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Stand-in structure for view `device_earnings_summary`
-- (See below for the actual view)
--
CREATE TABLE `device_earnings_summary` (
`deviceID` varchar(50)
,`deviceName` varchar(50)
,`numJobs` bigint
,`totalEarnings` decimal(51,2)
);

-- --------------------------------------------------------

--
-- Table structure for table `digital_signatures`
--

CREATE TABLE `digital_signatures` (
  `signatureID` int NOT NULL,
  `documentID` int NOT NULL,
  `signer_name` varchar(100) NOT NULL,
  `signer_email` varchar(100) DEFAULT NULL,
  `signer_role` varchar(50) DEFAULT NULL,
  `signature_data` longtext NOT NULL,
  `signed_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `ip_address` varchar(45) DEFAULT NULL,
  `verification_code` varchar(100) DEFAULT NULL,
  `is_verified` tinyint(1) DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `documents`
--

CREATE TABLE `documents` (
  `documentID` int NOT NULL,
  `entity_type` enum('job','device','customer','user','system') NOT NULL,
  `entity_id` varchar(50) NOT NULL,
  `filename` varchar(255) NOT NULL,
  `original_filename` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `file_size` bigint NOT NULL,
  `mime_type` varchar(100) NOT NULL,
  `document_type` enum('contract','manual','photo','invoice','receipt','signature','other') NOT NULL,
  `description` text,
  `uploaded_by` bigint UNSIGNED DEFAULT NULL,
  `uploaded_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `is_public` tinyint(1) DEFAULT '0',
  `version` int DEFAULT '1',
  `parent_documentID` int DEFAULT NULL,
  `checksum` varchar(64) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `document_signatures`
--

CREATE TABLE `document_signatures` (
  `id` bigint UNSIGNED NOT NULL,
  `document_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `document_id` bigint UNSIGNED NOT NULL,
  `content_hash` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `signature_data` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `algorithm` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'RSA-SHA256',
  `public_key` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `signed_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `signer_id` bigint UNSIGNED DEFAULT NULL,
  `verification_status` enum('valid','invalid','pending') COLLATE utf8mb4_unicode_ci DEFAULT 'pending',
  `last_verified_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `email_templates`
--

CREATE TABLE `email_templates` (
  `template_id` int UNSIGNED NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `template_type` enum('invoice','reminder','payment_confirmation','general') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'general',
  `subject` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL,
  `html_content` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `text_content` longtext COLLATE utf8mb4_unicode_ci,
  `is_default` tinyint(1) NOT NULL DEFAULT '0',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_by` int UNSIGNED DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `employee`
--

CREATE TABLE `employee` (
  `employeeID` int NOT NULL,
  `firstname` varchar(50) NOT NULL,
  `lastname` varchar(50) NOT NULL,
  `street` varchar(100) DEFAULT NULL,
  `housenumber` varchar(20) DEFAULT NULL,
  `ZIP` varchar(20) DEFAULT NULL,
  `city` varchar(50) DEFAULT NULL,
  `federalstate` varchar(50) DEFAULT NULL,
  `country` varchar(50) DEFAULT NULL,
  `phonenumber` varchar(20) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `employeejob`
--

CREATE TABLE `employeejob` (
  `employeeID` int NOT NULL,
  `jobID` int NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `encrypted_personal_data`
--

CREATE TABLE `encrypted_personal_data` (
  `id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `data_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `encrypted_data` longtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `key_version` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `algorithm` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `equipment_packages`
--

CREATE TABLE `equipment_packages` (
  `packageID` int NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` text,
  `categoryID` int DEFAULT NULL,
  `package_items` json NOT NULL,
  `package_price` decimal(12,2) DEFAULT NULL,
  `discount_percent` decimal(5,2) DEFAULT '0.00',
  `min_rental_days` int DEFAULT '1',
  `is_active` tinyint(1) DEFAULT '1',
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `usage_count` int DEFAULT '0',
  `max_rental_days` int DEFAULT NULL,
  `category` varchar(50) DEFAULT NULL,
  `tags` text,
  `last_used_at` timestamp NULL DEFAULT NULL,
  `total_revenue` decimal(12,2) DEFAULT '0.00'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `equipment_usage_logs`
--

CREATE TABLE `equipment_usage_logs` (
  `logID` int NOT NULL,
  `deviceID` varchar(50) NOT NULL,
  `jobID` int DEFAULT NULL,
  `action` enum('assigned','returned','maintenance','available') NOT NULL,
  `timestamp` datetime NOT NULL,
  `duration_hours` decimal(10,2) DEFAULT NULL,
  `revenue_generated` decimal(12,2) DEFAULT NULL,
  `notes` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `financial_transactions`
--

CREATE TABLE `financial_transactions` (
  `transactionID` int NOT NULL,
  `jobID` int DEFAULT NULL,
  `customerID` int DEFAULT NULL,
  `type` enum('rental','deposit','payment','refund','fee','discount') NOT NULL,
  `amount` decimal(12,2) NOT NULL,
  `currency` varchar(3) DEFAULT 'EUR',
  `status` enum('pending','completed','failed','cancelled') NOT NULL,
  `payment_method` varchar(50) DEFAULT NULL,
  `transaction_date` datetime NOT NULL,
  `due_date` date DEFAULT NULL,
  `reference_number` varchar(100) DEFAULT NULL,
  `notes` text,
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `gobd_records`
--

CREATE TABLE `gobd_records` (
  `id` bigint UNSIGNED NOT NULL,
  `document_type` varchar(191) NOT NULL,
  `document_id` varchar(191) NOT NULL,
  `original_data` longtext,
  `data_hash` varchar(191) NOT NULL,
  `archive_date` datetime(3) NOT NULL,
  `retention_date` datetime(3) NOT NULL,
  `digital_sign` text,
  `user_id` bigint UNSIGNED DEFAULT NULL,
  `company_id` bigint UNSIGNED DEFAULT NULL,
  `is_immutable` tinyint(1) DEFAULT '1',
  `archive_file_name` longtext,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `insuranceprovider`
--

CREATE TABLE `insuranceprovider` (
  `insuranceproviderID` int NOT NULL,
  `name` varchar(20) NOT NULL,
  `website` varchar(20) NOT NULL,
  `phonenumber` varchar(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `insurances`
--

CREATE TABLE `insurances` (
  `insuranceID` int NOT NULL,
  `name` varchar(20) NOT NULL,
  `insuranceproviderID` int NOT NULL,
  `policynumber` varchar(50) DEFAULT NULL,
  `coveragedetails` text,
  `validuntil` date DEFAULT NULL,
  `price` decimal(10,2) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoices`
--

CREATE TABLE `invoices` (
  `invoice_id` bigint UNSIGNED NOT NULL,
  `invoice_number` varchar(50) NOT NULL,
  `customer_id` int NOT NULL,
  `job_id` int DEFAULT NULL,
  `template_id` int DEFAULT NULL,
  `status` enum('draft','sent','paid','overdue','cancelled') NOT NULL DEFAULT 'draft',
  `issue_date` date NOT NULL,
  `due_date` date NOT NULL,
  `payment_terms` varchar(100) DEFAULT NULL,
  `subtotal` decimal(12,2) NOT NULL DEFAULT '0.00',
  `tax_rate` decimal(5,2) NOT NULL DEFAULT '0.00',
  `tax_amount` decimal(12,2) NOT NULL DEFAULT '0.00',
  `discount_amount` decimal(12,2) NOT NULL DEFAULT '0.00',
  `total_amount` decimal(12,2) NOT NULL DEFAULT '0.00',
  `paid_amount` decimal(12,2) NOT NULL DEFAULT '0.00',
  `balance_due` decimal(12,2) NOT NULL DEFAULT '0.00',
  `notes` text,
  `terms_conditions` text,
  `internal_notes` text,
  `sent_at` timestamp NULL DEFAULT NULL,
  `paid_at` timestamp NULL DEFAULT NULL,
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_line_items`
--

CREATE TABLE `invoice_line_items` (
  `line_item_id` bigint UNSIGNED NOT NULL,
  `invoice_id` bigint UNSIGNED NOT NULL,
  `item_type` enum('device','service','package','custom') NOT NULL DEFAULT 'custom',
  `device_id` varchar(50) DEFAULT NULL,
  `package_id` int DEFAULT NULL,
  `description` text NOT NULL,
  `quantity` decimal(10,2) NOT NULL DEFAULT '1.00',
  `unit_price` decimal(12,2) NOT NULL DEFAULT '0.00',
  `total_price` decimal(12,2) NOT NULL DEFAULT '0.00',
  `rental_start_date` date DEFAULT NULL,
  `rental_end_date` date DEFAULT NULL,
  `rental_days` int DEFAULT NULL,
  `sort_order` int UNSIGNED DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_payments`
--

CREATE TABLE `invoice_payments` (
  `payment_id` bigint UNSIGNED NOT NULL,
  `invoice_id` bigint UNSIGNED NOT NULL,
  `amount` decimal(12,2) NOT NULL,
  `payment_method` varchar(100) DEFAULT NULL,
  `payment_date` date NOT NULL,
  `reference_number` varchar(100) DEFAULT NULL,
  `notes` text,
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_settings`
--

CREATE TABLE `invoice_settings` (
  `setting_id` int NOT NULL,
  `setting_key` varchar(100) NOT NULL,
  `setting_value` text,
  `setting_type` enum('text','number','boolean','json') NOT NULL DEFAULT 'text',
  `description` text,
  `updated_by` bigint UNSIGNED DEFAULT NULL,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_templates`
--

CREATE TABLE `invoice_templates` (
  `template_id` int NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` text,
  `html_template` longtext NOT NULL,
  `css_styles` longtext,
  `is_default` tinyint(1) NOT NULL DEFAULT '0',
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `jobCategory`
--

CREATE TABLE `jobCategory` (
  `jobcategoryID` int NOT NULL,
  `name` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `abbreviation` varchar(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `jobdevices`
--

CREATE TABLE `jobdevices` (
  `jobID` int NOT NULL,
  `deviceID` varchar(50) NOT NULL,
  `custom_price` decimal(10,2) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `jobs`
--

CREATE TABLE `jobs` (
  `jobID` int NOT NULL,
  `customerID` int DEFAULT NULL,
  `startDate` date DEFAULT NULL,
  `endDate` date DEFAULT NULL,
  `statusID` int DEFAULT NULL,
  `jobcategoryID` int DEFAULT NULL,
  `description` varchar(50) DEFAULT NULL,
  `discount` decimal(10,2) DEFAULT '0.00',
  `discount_type` enum('percent','amount') DEFAULT 'amount',
  `revenue` decimal(12,2) NOT NULL DEFAULT '0.00' COMMENT 'Tatsächliche Einnahmen des Jobs in EUR',
  `final_revenue` decimal(10,2) DEFAULT NULL COMMENT 'Netto-Umsatz nach Rabatt',
  `priority` enum('low','normal','high','urgent') DEFAULT 'normal',
  `internal_notes` text,
  `customer_notes` text,
  `estimated_revenue` decimal(12,2) DEFAULT NULL,
  `actual_cost` decimal(12,2) DEFAULT '0.00',
  `profit_margin` decimal(5,2) DEFAULT NULL,
  `contract_signed` tinyint(1) DEFAULT '0',
  `contract_documentID` int DEFAULT NULL,
  `completion_percentage` int DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `jobs`
--
DELIMITER $$
CREATE TRIGGER `jobs_before_insert` BEFORE INSERT ON `jobs` FOR EACH ROW BEGIN
  IF NEW.discount_type = 'percent' THEN
    -- Prozentualer Rabatt
    SET NEW.final_revenue = ROUND(
      NEW.revenue * (1 - NEW.discount/100),
      2
    );
  ELSE
    -- Fixer Betrag
    SET NEW.final_revenue = ROUND(
      GREATEST(NEW.revenue - NEW.discount, 0),
      2
    );
  END IF;
END
$$
DELIMITER ;
DELIMITER $$
CREATE TRIGGER `jobs_before_update` BEFORE UPDATE ON `jobs` FOR EACH ROW BEGIN
  IF NEW.discount_type = 'percent' THEN
    SET NEW.final_revenue = ROUND(
      NEW.revenue * (1 - NEW.discount/100),
      2
    );
  ELSE
    SET NEW.final_revenue = ROUND(
      GREATEST(NEW.revenue - NEW.discount, 0),
      2
    );
  END IF;
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Table structure for table `maintenanceLogs`
--

CREATE TABLE `maintenanceLogs` (
  `maintenanceLogID` int NOT NULL,
  `deviceID` varchar(50) DEFAULT NULL,
  `date` datetime DEFAULT NULL,
  `employeeID` int DEFAULT NULL,
  `cost` decimal(10,2) DEFAULT NULL,
  `notes` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `manufacturer`
--

CREATE TABLE `manufacturer` (
  `manufacturerID` int NOT NULL,
  `name` varchar(100) NOT NULL,
  `website` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `offline_sync_queue`
--

CREATE TABLE `offline_sync_queue` (
  `queueID` int NOT NULL,
  `userID` bigint UNSIGNED NOT NULL,
  `action` enum('create','update','delete') NOT NULL,
  `entity_type` varchar(50) NOT NULL,
  `entity_data` json NOT NULL,
  `timestamp` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `synced` tinyint(1) DEFAULT '0',
  `synced_at` timestamp NULL DEFAULT NULL,
  `retry_count` int DEFAULT '0',
  `error_message` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `package_categories`
--

CREATE TABLE `package_categories` (
  `categoryID` int NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` text,
  `color` varchar(7) DEFAULT NULL COMMENT 'Hex color code for UI (#007bff)',
  `sort_order` int UNSIGNED DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `package_devices`
--

CREATE TABLE `package_devices` (
  `packageID` int NOT NULL,
  `deviceID` varchar(50) NOT NULL,
  `quantity` int UNSIGNED NOT NULL DEFAULT '1',
  `custom_price` decimal(12,2) DEFAULT NULL COMMENT 'Override price for this device in package',
  `is_required` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Whether device is required (1) or optional (0)',
  `notes` text COMMENT 'Special notes about this device in package',
  `sort_order` int UNSIGNED DEFAULT NULL COMMENT 'Display order within package',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `products`
--

CREATE TABLE `products` (
  `productID` int NOT NULL,
  `name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `categoryID` int DEFAULT NULL,
  `subcategoryID` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `subbiercategoryID` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `manufacturerID` int DEFAULT NULL,
  `brandID` int DEFAULT NULL,
  `description` text,
  `maintenanceInterval` int DEFAULT NULL,
  `itemcostperday` decimal(10,2) DEFAULT NULL COMMENT 'in €',
  `weight` decimal(10,2) DEFAULT NULL COMMENT 'in kg',
  `height` decimal(10,2) DEFAULT NULL COMMENT 'in cm',
  `width` decimal(10,2) DEFAULT NULL COMMENT 'in cm',
  `depth` decimal(10,2) DEFAULT NULL COMMENT 'in cm',
  `powerconsumption` decimal(10,2) DEFAULT NULL COMMENT 'in W',
  `pos_in_category` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `products`
--
DELIMITER $$
CREATE TRIGGER `pos_in_subcategory` BEFORE INSERT ON `products` FOR EACH ROW BEGIN
  DECLARE next_pos INT;

  -- Ermittele die höchste bereits vergebene Position in dieser Subkategorie
  SELECT COALESCE(MAX(pos_in_category), 0) + 1
    INTO next_pos
    FROM products
   WHERE subcategoryID = NEW.subcategoryID;

  -- Setze das neue pos_in_category-Feld
  SET NEW.pos_in_category = next_pos;
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Stand-in structure for view `product_revenue`
-- (See below for the actual view)
--
CREATE TABLE `product_revenue` (
`product_name` varchar(50)
,`total_revenue` decimal(32,2)
);

-- --------------------------------------------------------

--
-- Table structure for table `push_subscriptions`
--

CREATE TABLE `push_subscriptions` (
  `subscriptionID` int NOT NULL,
  `userID` bigint UNSIGNED NOT NULL,
  `endpoint` text NOT NULL,
  `keys_p256dh` text NOT NULL,
  `keys_auth` text NOT NULL,
  `user_agent` text,
  `device_type` varchar(50) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `last_used` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `is_active` tinyint(1) DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `retention_policies`
--

CREATE TABLE `retention_policies` (
  `id` bigint UNSIGNED NOT NULL,
  `data_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `retention_period_days` int UNSIGNED NOT NULL,
  `legal_basis` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL,
  `auto_delete` tinyint(1) DEFAULT '0',
  `policy_description` text COLLATE utf8mb4_unicode_ci,
  `effective_from` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `effective_until` timestamp NULL DEFAULT NULL,
  `created_by` bigint UNSIGNED DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- --------------------------------------------------------

--
-- Table structure for table `roles`
--

CREATE TABLE `roles` (
  `roleID` int NOT NULL,
  `name` varchar(50) NOT NULL,
  `display_name` varchar(100) NOT NULL,
  `description` text,
  `permissions` json NOT NULL,
  `is_system_role` tinyint(1) DEFAULT '0',
  `is_active` tinyint(1) DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `saved_searches`
--

CREATE TABLE `saved_searches` (
  `searchID` int NOT NULL,
  `userID` bigint UNSIGNED NOT NULL,
  `name` varchar(100) NOT NULL,
  `search_type` enum('global','jobs','devices','customers','cases') NOT NULL,
  `filters` json NOT NULL,
  `is_default` tinyint(1) DEFAULT '0',
  `is_public` tinyint(1) DEFAULT '0',
  `usage_count` int DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `last_used` timestamp NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `search_history`
--

CREATE TABLE `search_history` (
  `historyID` int NOT NULL,
  `userID` bigint UNSIGNED DEFAULT NULL,
  `search_term` varchar(500) DEFAULT NULL,
  `search_type` varchar(50) DEFAULT NULL,
  `filters` json DEFAULT NULL,
  `results_count` int DEFAULT NULL,
  `execution_time_ms` int DEFAULT NULL,
  `searched_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `sessions`
--

CREATE TABLE `sessions` (
  `session_id` varchar(191) NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `status`
--

CREATE TABLE `status` (
  `statusID` int NOT NULL,
  `status` varchar(11) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `subbiercategories`
--

CREATE TABLE `subbiercategories` (
  `subbiercategoryID` varchar(50) NOT NULL,
  `name` varchar(20) DEFAULT NULL,
  `abbreviation` varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `subcategoryID` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `subbiercategories`
--
DELIMITER $$
CREATE TRIGGER `before_insert_subbiercategory` BEFORE INSERT ON `subbiercategories` FOR EACH ROW BEGIN
    DECLARE subcat_abkuerzung VARCHAR(50);
    DECLARE naechste_nummer INT;
    
    -- Abkürzung aus der subcategories-Tabelle abrufen
    SELECT s.abbreviation INTO subcat_abkuerzung
    FROM subcategories s
    WHERE s.subcategoryID = NEW.subcategoryID
    LIMIT 1;
    
    -- Nächste verfügbare Nummer für diese Abkürzung finden
    SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(subbiercategoryID, subcat_abkuerzung, -1) AS UNSIGNED)), 1000) + 1 
    INTO naechste_nummer
    FROM subbiercategories sb
    JOIN subcategories s ON sb.subcategoryID = s.subcategoryID
    WHERE s.abbreviation = subcat_abkuerzung;
    
    -- subbiercategoryID setzen als Kombination aus Unterkategorie-Abkürzung und nächster Nummer
    SET NEW.subbiercategoryID = CONCAT(subcat_abkuerzung, naechste_nummer);
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Table structure for table `subcategories`
--

CREATE TABLE `subcategories` (
  `subcategoryID` varchar(50) NOT NULL,
  `name` varchar(20) NOT NULL,
  `abbreviation` varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `categoryID` int DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `subcategories`
--
DELIMITER $$
CREATE TRIGGER `before_insert_subcategory` BEFORE INSERT ON `subcategories` FOR EACH ROW BEGIN
    DECLARE cat_abkuerzung VARCHAR(50);
    DECLARE naechste_nummer INT;
    
    -- Abkürzung aus der categories-Tabelle abrufen
    SELECT c.abbreviation INTO cat_abkuerzung
    FROM categories c
    WHERE c.categoryID = NEW.categoryID
    LIMIT 1;
    
    -- Nächste verfügbare Nummer für diese Abkürzung finden
    SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(subcategoryID, cat_abkuerzung, -1) AS UNSIGNED)), 1000) + 1 
    INTO naechste_nummer
    FROM subcategories s
    JOIN categories c ON s.categoryID = c.categoryID
    WHERE c.abbreviation = cat_abkuerzung;
    
    -- subcategoryID setzen als Kombination aus Kategorie-Abkürzung und nächster Nummer
    SET NEW.subcategoryID = CONCAT(cat_abkuerzung, naechste_nummer);
END
$$
DELIMITER ;

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE `users` (
  `userID` bigint UNSIGNED NOT NULL,
  `username` varchar(191) NOT NULL,
  `email` varchar(191) NOT NULL,
  `password_hash` longtext NOT NULL,
  `first_name` longtext,
  `last_name` longtext,
  `is_active` tinyint(1) DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `last_login` datetime(3) DEFAULT NULL,
  `timezone` varchar(50) DEFAULT 'Europe/Berlin',
  `language` varchar(5) DEFAULT 'en',
  `avatar_path` varchar(500) DEFAULT NULL,
  `notification_preferences` json DEFAULT NULL,
  `last_active` timestamp NULL DEFAULT NULL,
  `login_attempts` int DEFAULT '0',
  `locked_until` timestamp NULL DEFAULT NULL,
  `two_factor_enabled` tinyint(1) DEFAULT '0',
  `two_factor_secret` varchar(100) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_preferences`
--

CREATE TABLE `user_preferences` (
  `preference_id` bigint UNSIGNED NOT NULL,
  `user_id` bigint UNSIGNED NOT NULL,
  `language` varchar(191) NOT NULL DEFAULT 'de',
  `theme` varchar(191) NOT NULL DEFAULT 'dark',
  `time_zone` varchar(191) NOT NULL DEFAULT 'Europe/Berlin',
  `date_format` varchar(191) NOT NULL DEFAULT 'DD.MM.YYYY',
  `time_format` varchar(191) NOT NULL DEFAULT '24h',
  `email_notifications` tinyint(1) NOT NULL DEFAULT '1',
  `system_notifications` tinyint(1) NOT NULL DEFAULT '1',
  `job_status_notifications` tinyint(1) NOT NULL DEFAULT '1',
  `device_alert_notifications` tinyint(1) NOT NULL DEFAULT '1',
  `items_per_page` bigint NOT NULL DEFAULT '25',
  `default_view` varchar(191) NOT NULL DEFAULT 'list',
  `show_advanced_options` tinyint(1) NOT NULL DEFAULT '0',
  `auto_save_enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_roles`
--

CREATE TABLE `user_roles` (
  `userID` bigint UNSIGNED NOT NULL,
  `roleID` int NOT NULL,
  `assigned_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `assigned_by` bigint UNSIGNED DEFAULT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '1'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- --------------------------------------------------------

--
-- Stand-in structure for view `view_device_product`
-- (See below for the actual view)
--
CREATE TABLE `view_device_product` (
`deviceID` varchar(50)
,`product_name` varchar(50)
,`productID` int
);

-- --------------------------------------------------------

--
-- Stand-in structure for view `vw_cable_overview`
-- (See below for the actual view)
--
CREATE TABLE `vw_cable_overview` (
`cable_name` varchar(100)
,`length_display` varchar(14)
);

-- --------------------------------------------------------

--
-- Stand-in structure for view `vw_invoice_summary`
-- (See below for the actual view)
--
CREATE TABLE `vw_invoice_summary` (
`balance_due` decimal(12,2)
,`customer_id` int
,`customer_name` varchar(101)
,`days_overdue` int
,`due_date` date
,`invoice_id` bigint unsigned
,`invoice_number` varchar(50)
,`issue_date` date
,`item_count` bigint
,`job_description` varchar(50)
,`job_id` int
,`paid_amount` decimal(12,2)
,`status` enum('draft','sent','paid','overdue','cancelled')
,`total_amount` decimal(12,2)
);

-- --------------------------------------------------------

--
-- Stand-in structure for view `vw_package_devices_detail`
-- (See below for the actual view)
--
CREATE TABLE `vw_package_devices_detail` (
`custom_price` decimal(12,2)
,`defaultPrice` decimal(10,2)
,`deviceID` varchar(50)
,`deviceStatus` varchar(50)
,`effectivePrice` decimal(12,2)
,`is_required` tinyint(1)
,`lineTotal` decimal(22,2)
,`notes` text
,`packageID` int
,`packageName` varchar(100)
,`productCategory` varchar(43)
,`productName` varchar(50)
,`quantity` int unsigned
,`serialNumber` varchar(50)
,`sort_order` int unsigned
);

-- --------------------------------------------------------

--
-- Stand-in structure for view `vw_package_summary`
-- (See below for the actual view)
--
CREATE TABLE `vw_package_summary` (
`categoryColor` varchar(7)
,`categoryName` varchar(100)
,`createdAt` timestamp
,`description` text
,`deviceCount` bigint
,`discountPercent` decimal(5,2)
,`isActive` tinyint(1)
,`minRentalDays` int
,`optionalDevices` decimal(31,0)
,`packageID` int
,`packageName` varchar(100)
,`packagePrice` decimal(12,2)
,`requiredDevices` decimal(31,0)
,`totalDevices` decimal(32,0)
,`updatedAt` timestamp
,`usageCount` int
);

--
-- Indexes for dumped tables
--

--
-- Indexes for table `analytics_cache`
--
ALTER TABLE `analytics_cache`
  ADD PRIMARY KEY (`cacheID`),
  ADD UNIQUE KEY `unique_metric` (`metric_name`,`period_type`,`period_date`),
  ADD KEY `idx_metric_period` (`metric_name`,`period_type`);

--
-- Indexes for table `archived_documents`
--
ALTER TABLE `archived_documents`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_document` (`document_type`,`document_id`),
  ADD KEY `idx_archived_docs_type` (`document_type`),
  ADD KEY `idx_archived_docs_retention` (`retention_until`),
  ADD KEY `idx_archived_docs_hash` (`original_hash`),
  ADD KEY `idx_archived_documents_legal` (`legal_basis`);

--
-- Indexes for table `audit_events`
--
ALTER TABLE `audit_events`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `event_hash` (`event_hash`),
  ADD KEY `idx_audit_events_timestamp` (`timestamp`),
  ADD KEY `idx_audit_events_entity` (`entity_type`,`entity_id`),
  ADD KEY `idx_audit_events_user` (`user_id`),
  ADD KEY `idx_audit_events_object_type` (`object_type`),
  ADD KEY `idx_audit_events_object_id` (`object_id`),
  ADD KEY `idx_audit_events_user_id` (`user_id`),
  ADD KEY `idx_audit_events_session_id` (`session_id`),
  ADD KEY `idx_audit_events_previous_hash` (`previous_hash`),
  ADD KEY `idx_audit_events_retention_date` (`retention_date`),
  ADD KEY `idx_audit_events_event_type` (`event_type`);

--
-- Indexes for table `audit_log`
--
ALTER TABLE `audit_log`
  ADD PRIMARY KEY (`auditID`),
  ADD KEY `idx_entity` (`entity_type`,`entity_id`),
  ADD KEY `idx_user_time` (`userID`,`timestamp`),
  ADD KEY `idx_action_time` (`action`,`timestamp`),
  ADD KEY `idx_timestamp` (`timestamp`);

--
-- Indexes for table `audit_logs`
--
ALTER TABLE `audit_logs`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_audit_logs_entity` (`entity_type`,`entity_id`),
  ADD KEY `idx_audit_logs_user` (`user_id`),
  ADD KEY `idx_audit_logs_timestamp` (`timestamp`),
  ADD KEY `idx_audit_logs_hash` (`hash`),
  ADD KEY `idx_audit_logs_chain` (`previous_hash`,`hash`);

--
-- Indexes for table `brands`
--
ALTER TABLE `brands`
  ADD PRIMARY KEY (`brandID`),
  ADD KEY `idx_brands_manufacturerID` (`manufacturerID`);

--
-- Indexes for table `cables`
--
ALTER TABLE `cables`
  ADD PRIMARY KEY (`cableID`),
  ADD KEY `connector1` (`connector1`),
  ADD KEY `connector2` (`connector2`),
  ADD KEY `typ` (`typ`);

--
-- Indexes for table `cable_connectors`
--
ALTER TABLE `cable_connectors`
  ADD PRIMARY KEY (`cable_connectorsID`);

--
-- Indexes for table `cable_types`
--
ALTER TABLE `cable_types`
  ADD PRIMARY KEY (`cable_typesID`);

--
-- Indexes for table `cases`
--
ALTER TABLE `cases`
  ADD PRIMARY KEY (`caseID`);

--
-- Indexes for table `categories`
--
ALTER TABLE `categories`
  ADD PRIMARY KEY (`categoryID`);

--
-- Indexes for table `company_settings`
--
ALTER TABLE `company_settings`
  ADD PRIMARY KEY (`id`);

--
-- Indexes for table `consent_records`
--
ALTER TABLE `consent_records`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_consent_user` (`user_id`),
  ADD KEY `idx_consent_type_purpose` (`data_type`,`purpose`),
  ADD KEY `idx_consent_date` (`consent_date`),
  ADD KEY `idx_consent_expiry` (`expiry_date`);

--
-- Indexes for table `customers`
--
ALTER TABLE `customers`
  ADD PRIMARY KEY (`customerID`),
  ADD KEY `idx_customers_search_company` (`companyname`),
  ADD KEY `idx_customers_search_name` (`firstname`,`lastname`),
  ADD KEY `idx_customers_email` (`email`);
ALTER TABLE `customers` ADD FULLTEXT KEY `idx_customers_search` (`companyname`,`firstname`,`lastname`,`email`);

--
-- Indexes for table `data_processing_records`
--
ALTER TABLE `data_processing_records`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_data_processing_user` (`user_id`),
  ADD KEY `idx_data_processing_type` (`data_type`),
  ADD KEY `idx_data_processing_purpose` (`purpose`),
  ADD KEY `idx_data_processing_expiry` (`expires_at`);

--
-- Indexes for table `data_subject_requests`
--
ALTER TABLE `data_subject_requests`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_data_subject_user` (`user_id`),
  ADD KEY `idx_data_subject_type` (`request_type`),
  ADD KEY `idx_data_subject_status` (`status`),
  ADD KEY `idx_data_subject_requested` (`requested_at`);

--
-- Indexes for table `devices`
--
ALTER TABLE `devices`
  ADD PRIMARY KEY (`deviceID`),
  ADD UNIQUE KEY `qr_code` (`qr_code`),
  ADD KEY `idx_devices_insuranceID` (`insuranceID`),
  ADD KEY `idx_devices_productID` (`productID`),
  ADD KEY `idx_devices_location` (`current_location`),
  ADD KEY `idx_devices_qr` (`qr_code`),
  ADD KEY `idx_devices_status` (`status`),
  ADD KEY `idx_devices_search` (`deviceID`,`serialnumber`),
  ADD KEY `idx_devices_product_status` (`productID`,`status`);

--
-- Indexes for table `devicescases`
--
ALTER TABLE `devicescases`
  ADD PRIMARY KEY (`caseID`,`deviceID`),
  ADD KEY `deviceID` (`deviceID`);

--
-- Indexes for table `devicestatushistory`
--
ALTER TABLE `devicestatushistory`
  ADD PRIMARY KEY (`statushistoryID`),
  ADD KEY `idx_devicestatushistory_deviceID` (`deviceID`);

--
-- Indexes for table `digital_signatures`
--
ALTER TABLE `digital_signatures`
  ADD PRIMARY KEY (`signatureID`),
  ADD KEY `idx_document_signer` (`documentID`,`signer_email`),
  ADD KEY `idx_signed_date` (`signed_at`);

--
-- Indexes for table `documents`
--
ALTER TABLE `documents`
  ADD PRIMARY KEY (`documentID`),
  ADD KEY `uploaded_by` (`uploaded_by`),
  ADD KEY `parent_documentID` (`parent_documentID`),
  ADD KEY `idx_entity_type` (`entity_type`,`entity_id`,`document_type`),
  ADD KEY `idx_uploaded_date` (`uploaded_at`,`document_type`),
  ADD KEY `idx_filename` (`filename`),
  ADD KEY `idx_documents_entity` (`entity_type`,`entity_id`,`document_type`),
  ADD KEY `idx_documents_date` (`uploaded_at`,`document_type`);

--
-- Indexes for table `document_signatures`
--
ALTER TABLE `document_signatures`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_document_signature` (`document_type`,`document_id`),
  ADD KEY `idx_doc_signatures_type` (`document_type`),
  ADD KEY `idx_doc_signatures_signer` (`signer_id`),
  ADD KEY `idx_doc_signatures_status` (`verification_status`),
  ADD KEY `idx_document_signatures_hash` (`content_hash`);

--
-- Indexes for table `email_templates`
--
ALTER TABLE `email_templates`
  ADD PRIMARY KEY (`template_id`),
  ADD KEY `idx_email_templates_type` (`template_type`),
  ADD KEY `idx_email_templates_default` (`is_default`),
  ADD KEY `idx_email_templates_active` (`is_active`),
  ADD KEY `idx_email_templates_created_by` (`created_by`);

--
-- Indexes for table `employee`
--
ALTER TABLE `employee`
  ADD PRIMARY KEY (`employeeID`);

--
-- Indexes for table `employeejob`
--
ALTER TABLE `employeejob`
  ADD PRIMARY KEY (`employeeID`,`jobID`),
  ADD KEY `idx_employeejob_jobID` (`jobID`);

--
-- Indexes for table `encrypted_personal_data`
--
ALTER TABLE `encrypted_personal_data`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_user_data_type` (`user_id`,`data_type`),
  ADD KEY `idx_encrypted_data_user` (`user_id`),
  ADD KEY `idx_encrypted_data_type` (`data_type`),
  ADD KEY `idx_encrypted_data_key_version` (`key_version`);

--
-- Indexes for table `equipment_packages`
--
ALTER TABLE `equipment_packages`
  ADD PRIMARY KEY (`packageID`),
  ADD KEY `created_by` (`created_by`),
  ADD KEY `idx_active_usage` (`is_active`,`usage_count` DESC),
  ADD KEY `idx_equipment_packages_category` (`categoryID`);

--
-- Indexes for table `equipment_usage_logs`
--
ALTER TABLE `equipment_usage_logs`
  ADD PRIMARY KEY (`logID`),
  ADD KEY `idx_device_timestamp` (`deviceID`,`timestamp`),
  ADD KEY `idx_job_action` (`jobID`,`action`),
  ADD KEY `idx_timestamp_action` (`timestamp`,`action`),
  ADD KEY `idx_usage_logs_device_date` (`deviceID`,`timestamp`);

--
-- Indexes for table `financial_transactions`
--
ALTER TABLE `financial_transactions`
  ADD PRIMARY KEY (`transactionID`),
  ADD KEY `jobID` (`jobID`),
  ADD KEY `created_by` (`created_by`),
  ADD KEY `idx_customer_date` (`customerID`,`transaction_date`),
  ADD KEY `idx_status_due` (`status`,`due_date`),
  ADD KEY `idx_type_date` (`type`,`transaction_date`),
  ADD KEY `idx_transactions_customer_date` (`customerID`,`transaction_date`),
  ADD KEY `idx_transactions_status` (`status`,`due_date`);

--
-- Indexes for table `gobd_records`
--
ALTER TABLE `gobd_records`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_gobd_records_data_hash` (`data_hash`),
  ADD KEY `idx_gobd_records_archive_date` (`archive_date`),
  ADD KEY `idx_gobd_records_retention_date` (`retention_date`),
  ADD KEY `idx_gobd_records_user_id` (`user_id`),
  ADD KEY `idx_gobd_records_company_id` (`company_id`),
  ADD KEY `idx_gobd_records_document_type` (`document_type`),
  ADD KEY `idx_gobd_records_document_id` (`document_id`);

--
-- Indexes for table `insuranceprovider`
--
ALTER TABLE `insuranceprovider`
  ADD PRIMARY KEY (`insuranceproviderID`);

--
-- Indexes for table `insurances`
--
ALTER TABLE `insurances`
  ADD PRIMARY KEY (`insuranceID`),
  ADD KEY `insuranceproviderID` (`insuranceproviderID`);

--
-- Indexes for table `invoices`
--
ALTER TABLE `invoices`
  ADD PRIMARY KEY (`invoice_id`),
  ADD UNIQUE KEY `invoice_number` (`invoice_number`),
  ADD KEY `idx_invoices_customer` (`customer_id`),
  ADD KEY `idx_invoices_job` (`job_id`),
  ADD KEY `idx_invoices_status` (`status`),
  ADD KEY `idx_invoices_issue_date` (`issue_date`),
  ADD KEY `idx_invoices_due_date` (`due_date`),
  ADD KEY `idx_invoices_number` (`invoice_number`),
  ADD KEY `fk_invoices_template` (`template_id`),
  ADD KEY `invoices_ibfk_1` (`created_by`);

--
-- Indexes for table `invoice_line_items`
--
ALTER TABLE `invoice_line_items`
  ADD PRIMARY KEY (`line_item_id`),
  ADD KEY `idx_invoice_line_items_invoice` (`invoice_id`),
  ADD KEY `idx_invoice_line_items_device` (`device_id`),
  ADD KEY `idx_invoice_line_items_package` (`package_id`),
  ADD KEY `idx_invoice_line_items_type` (`item_type`);

--
-- Indexes for table `invoice_payments`
--
ALTER TABLE `invoice_payments`
  ADD PRIMARY KEY (`payment_id`),
  ADD KEY `idx_invoice_payments_invoice` (`invoice_id`),
  ADD KEY `idx_invoice_payments_date` (`payment_date`),
  ADD KEY `invoice_payments_ibfk_2` (`created_by`);

--
-- Indexes for table `invoice_settings`
--
ALTER TABLE `invoice_settings`
  ADD PRIMARY KEY (`setting_id`),
  ADD UNIQUE KEY `setting_key` (`setting_key`),
  ADD KEY `idx_invoice_settings_key` (`setting_key`),
  ADD KEY `invoice_settings_ibfk_1` (`updated_by`);

--
-- Indexes for table `invoice_templates`
--
ALTER TABLE `invoice_templates`
  ADD PRIMARY KEY (`template_id`),
  ADD KEY `idx_invoice_templates_default` (`is_default`),
  ADD KEY `idx_invoice_templates_active` (`is_active`),
  ADD KEY `fk_invoice_templates_created_by` (`created_by`);

--
-- Indexes for table `jobCategory`
--
ALTER TABLE `jobCategory`
  ADD PRIMARY KEY (`jobcategoryID`);

--
-- Indexes for table `jobdevices`
--
ALTER TABLE `jobdevices`
  ADD PRIMARY KEY (`jobID`,`deviceID`),
  ADD KEY `deviceID` (`deviceID`),
  ADD KEY `idx_jobdevices_deviceid` (`deviceID`),
  ADD KEY `idx_jobdevices_jobid` (`jobID`),
  ADD KEY `idx_jobdevices_composite` (`deviceID`,`jobID`);

--
-- Indexes for table `jobs`
--
ALTER TABLE `jobs`
  ADD PRIMARY KEY (`jobID`),
  ADD KEY `idx_jobs_customerID` (`customerID`),
  ADD KEY `idx_jobs_jobcategoryID` (`jobcategoryID`),
  ADD KEY `statusID` (`statusID`),
  ADD KEY `contract_documentID` (`contract_documentID`),
  ADD KEY `idx_jobs_statusid` (`statusID`);
ALTER TABLE `jobs` ADD FULLTEXT KEY `idx_jobs_search` (`description`,`internal_notes`,`customer_notes`);

--
-- Indexes for table `maintenanceLogs`
--
ALTER TABLE `maintenanceLogs`
  ADD PRIMARY KEY (`maintenanceLogID`),
  ADD KEY `idx_maintenanceLogs_deviceID` (`deviceID`),
  ADD KEY `idx_maintenanceLogs_employeeID` (`employeeID`);

--
-- Indexes for table `manufacturer`
--
ALTER TABLE `manufacturer`
  ADD PRIMARY KEY (`manufacturerID`);

--
-- Indexes for table `offline_sync_queue`
--
ALTER TABLE `offline_sync_queue`
  ADD PRIMARY KEY (`queueID`),
  ADD KEY `idx_user_synced` (`userID`,`synced`),
  ADD KEY `idx_timestamp_synced` (`timestamp`,`synced`);

--
-- Indexes for table `package_categories`
--
ALTER TABLE `package_categories`
  ADD PRIMARY KEY (`categoryID`),
  ADD UNIQUE KEY `uk_package_categories_name` (`name`),
  ADD KEY `idx_package_categories_active` (`is_active`),
  ADD KEY `idx_package_categories_sort` (`sort_order`);

--
-- Indexes for table `package_devices`
--
ALTER TABLE `package_devices`
  ADD PRIMARY KEY (`packageID`,`deviceID`),
  ADD KEY `idx_package_devices_package` (`packageID`),
  ADD KEY `idx_package_devices_device` (`deviceID`),
  ADD KEY `idx_package_devices_required` (`is_required`),
  ADD KEY `idx_package_devices_sort` (`sort_order`);

--
-- Indexes for table `products`
--
ALTER TABLE `products`
  ADD PRIMARY KEY (`productID`),
  ADD KEY `idx_products_categoryID` (`categoryID`),
  ADD KEY `idx_products_manufacturerID` (`manufacturerID`),
  ADD KEY `idx_products_brandID` (`brandID`),
  ADD KEY `idx_products_subcategoryID` (`subcategoryID`),
  ADD KEY `idx_products_subbiercategoryID` (`subbiercategoryID`);

--
-- Indexes for table `push_subscriptions`
--
ALTER TABLE `push_subscriptions`
  ADD PRIMARY KEY (`subscriptionID`),
  ADD KEY `idx_user_active` (`userID`,`is_active`),
  ADD KEY `idx_last_used` (`last_used`);

--
-- Indexes for table `retention_policies`
--
ALTER TABLE `retention_policies`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_active_policy` (`data_type`,`effective_until`),
  ADD KEY `idx_retention_policies_type` (`data_type`),
  ADD KEY `idx_retention_policies_effective` (`effective_from`,`effective_until`);

--
-- Indexes for table `roles`
--
ALTER TABLE `roles`
  ADD PRIMARY KEY (`roleID`),
  ADD UNIQUE KEY `name` (`name`),
  ADD KEY `idx_active_system` (`is_active`,`is_system_role`);

--
-- Indexes for table `saved_searches`
--
ALTER TABLE `saved_searches`
  ADD PRIMARY KEY (`searchID`),
  ADD KEY `idx_user_type` (`userID`,`search_type`),
  ADD KEY `idx_usage_count` (`usage_count` DESC);

--
-- Indexes for table `search_history`
--
ALTER TABLE `search_history`
  ADD PRIMARY KEY (`historyID`),
  ADD KEY `idx_user_date` (`userID`,`searched_at`),
  ADD KEY `idx_search_type` (`search_type`,`searched_at`);

--
-- Indexes for table `sessions`
--
ALTER TABLE `sessions`
  ADD PRIMARY KEY (`session_id`);

--
-- Indexes for table `status`
--
ALTER TABLE `status`
  ADD PRIMARY KEY (`statusID`);

--
-- Indexes for table `subbiercategories`
--
ALTER TABLE `subbiercategories`
  ADD PRIMARY KEY (`subbiercategoryID`),
  ADD KEY `idx_subbiercategories_subcategoyID_unique` (`subcategoryID`) USING BTREE;

--
-- Indexes for table `subcategories`
--
ALTER TABLE `subcategories`
  ADD PRIMARY KEY (`subcategoryID`),
  ADD KEY `categoryID` (`categoryID`);

--
-- Indexes for table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`userID`),
  ADD UNIQUE KEY `username` (`username`),
  ADD UNIQUE KEY `email` (`email`);

--
-- Indexes for table `user_preferences`
--
ALTER TABLE `user_preferences`
  ADD PRIMARY KEY (`preference_id`),
  ADD UNIQUE KEY `user_id` (`user_id`);

--
-- Indexes for table `user_roles`
--
ALTER TABLE `user_roles`
  ADD PRIMARY KEY (`userID`,`roleID`),
  ADD KEY `assigned_by` (`assigned_by`),
  ADD KEY `idx_user_active` (`userID`,`is_active`),
  ADD KEY `idx_role_active` (`roleID`,`is_active`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `analytics_cache`
--
ALTER TABLE `analytics_cache`
  MODIFY `cacheID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `archived_documents`
--
ALTER TABLE `archived_documents`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `audit_events`
--
ALTER TABLE `audit_events`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `audit_log`
--
ALTER TABLE `audit_log`
  MODIFY `auditID` bigint NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `audit_logs`
--
ALTER TABLE `audit_logs`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `brands`
--
ALTER TABLE `brands`
  MODIFY `brandID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `cables`
--
ALTER TABLE `cables`
  MODIFY `cableID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `cable_connectors`
--
ALTER TABLE `cable_connectors`
  MODIFY `cable_connectorsID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `cable_types`
--
ALTER TABLE `cable_types`
  MODIFY `cable_typesID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `cases`
--
ALTER TABLE `cases`
  MODIFY `caseID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `categories`
--
ALTER TABLE `categories`
  MODIFY `categoryID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `company_settings`
--
ALTER TABLE `company_settings`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `consent_records`
--
ALTER TABLE `consent_records`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `customers`
--
ALTER TABLE `customers`
  MODIFY `customerID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `data_processing_records`
--
ALTER TABLE `data_processing_records`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `data_subject_requests`
--
ALTER TABLE `data_subject_requests`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `devicestatushistory`
--
ALTER TABLE `devicestatushistory`
  MODIFY `statushistoryID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `digital_signatures`
--
ALTER TABLE `digital_signatures`
  MODIFY `signatureID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `documents`
--
ALTER TABLE `documents`
  MODIFY `documentID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `document_signatures`
--
ALTER TABLE `document_signatures`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `email_templates`
--
ALTER TABLE `email_templates`
  MODIFY `template_id` int UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `employee`
--
ALTER TABLE `employee`
  MODIFY `employeeID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `encrypted_personal_data`
--
ALTER TABLE `encrypted_personal_data`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `equipment_packages`
--
ALTER TABLE `equipment_packages`
  MODIFY `packageID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `equipment_usage_logs`
--
ALTER TABLE `equipment_usage_logs`
  MODIFY `logID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `financial_transactions`
--
ALTER TABLE `financial_transactions`
  MODIFY `transactionID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `gobd_records`
--
ALTER TABLE `gobd_records`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `insuranceprovider`
--
ALTER TABLE `insuranceprovider`
  MODIFY `insuranceproviderID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `insurances`
--
ALTER TABLE `insurances`
  MODIFY `insuranceID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoices`
--
ALTER TABLE `invoices`
  MODIFY `invoice_id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_line_items`
--
ALTER TABLE `invoice_line_items`
  MODIFY `line_item_id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_payments`
--
ALTER TABLE `invoice_payments`
  MODIFY `payment_id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_settings`
--
ALTER TABLE `invoice_settings`
  MODIFY `setting_id` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_templates`
--
ALTER TABLE `invoice_templates`
  MODIFY `template_id` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `jobCategory`
--
ALTER TABLE `jobCategory`
  MODIFY `jobcategoryID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `jobs`
--
ALTER TABLE `jobs`
  MODIFY `jobID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `maintenanceLogs`
--
ALTER TABLE `maintenanceLogs`
  MODIFY `maintenanceLogID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `manufacturer`
--
ALTER TABLE `manufacturer`
  MODIFY `manufacturerID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `offline_sync_queue`
--
ALTER TABLE `offline_sync_queue`
  MODIFY `queueID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `package_categories`
--
ALTER TABLE `package_categories`
  MODIFY `categoryID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `products`
--
ALTER TABLE `products`
  MODIFY `productID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `push_subscriptions`
--
ALTER TABLE `push_subscriptions`
  MODIFY `subscriptionID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `retention_policies`
--
ALTER TABLE `retention_policies`
  MODIFY `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `roles`
--
ALTER TABLE `roles`
  MODIFY `roleID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `saved_searches`
--
ALTER TABLE `saved_searches`
  MODIFY `searchID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `search_history`
--
ALTER TABLE `search_history`
  MODIFY `historyID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `status`
--
ALTER TABLE `status`
  MODIFY `statusID` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `users`
--
ALTER TABLE `users`
  MODIFY `userID` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `user_preferences`
--
ALTER TABLE `user_preferences`
  MODIFY `preference_id` bigint UNSIGNED NOT NULL AUTO_INCREMENT;

-- --------------------------------------------------------

--
-- Structure for view `device_earnings_summary`
--
DROP TABLE IF EXISTS `device_earnings_summary`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `device_earnings_summary`  AS SELECT `d`.`deviceID` AS `deviceID`, `p`.`name` AS `deviceName`, count(distinct `jd`.`jobID`) AS `numJobs`, round(coalesce(sum((case when (`j`.`discount_type` = 'percent') then (coalesce(`jd`.`custom_price`,(((to_days(`j`.`endDate`) - to_days(`j`.`startDate`)) + 1) * `p`.`itemcostperday`)) * (1 - (`j`.`discount` / 100))) when (`j`.`discount_type` = 'amount') then greatest((coalesce(`jd`.`custom_price`,(((to_days(`j`.`endDate`) - to_days(`j`.`startDate`)) + 1) * `p`.`itemcostperday`)) - (`j`.`discount` / `jd_count`.`device_count`)),0) else coalesce(`jd`.`custom_price`,(((to_days(`j`.`endDate`) - to_days(`j`.`startDate`)) + 1) * `p`.`itemcostperday`)) end)),0),2) AS `totalEarnings` FROM ((((`devices` `d` left join `jobdevices` `jd` on((`d`.`deviceID` = `jd`.`deviceID`))) left join (select `jobdevices`.`jobID` AS `jobID`,count(0) AS `device_count` from `jobdevices` group by `jobdevices`.`jobID`) `jd_count` on((`jd`.`jobID` = `jd_count`.`jobID`))) left join `jobs` `j` on((`jd`.`jobID` = `j`.`jobID`))) left join `products` `p` on((`d`.`productID` = `p`.`productID`))) GROUP BY `d`.`deviceID`, `p`.`name` ;

-- --------------------------------------------------------

--
-- Structure for view `product_revenue`
--
DROP TABLE IF EXISTS `product_revenue`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `product_revenue`  AS SELECT `p`.`name` AS `product_name`, sum(`jd`.`custom_price`) AS `total_revenue` FROM (((`jobdevices` `jd` join `devices` `d` on((`jd`.`deviceID` = `d`.`deviceID`))) join `products` `p` on((`d`.`productID` = `p`.`productID`))) join `jobs` `j` on((`jd`.`jobID` = `j`.`jobID`))) GROUP BY `p`.`name` ORDER BY `total_revenue` DESC ;

-- --------------------------------------------------------

--
-- Structure for view `view_device_product`
--
DROP TABLE IF EXISTS `view_device_product`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `view_device_product`  AS SELECT `d`.`deviceID` AS `deviceID`, `p`.`name` AS `product_name`, `p`.`productID` AS `productID` FROM (`devices` `d` join `products` `p` on((`d`.`productID` = `p`.`productID`))) ;

-- --------------------------------------------------------

--
-- Structure for view `vw_cable_overview`
--
DROP TABLE IF EXISTS `vw_cable_overview`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `vw_cable_overview`  AS SELECT `cables`.`name` AS `cable_name`, concat(round(`cables`.`length`,2),' m') AS `length_display` FROM `cables` ;

-- --------------------------------------------------------

--
-- Structure for view `vw_invoice_summary`
--
DROP TABLE IF EXISTS `vw_invoice_summary`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `vw_invoice_summary`  AS SELECT `i`.`invoice_id` AS `invoice_id`, `i`.`invoice_number` AS `invoice_number`, `i`.`status` AS `status`, `i`.`issue_date` AS `issue_date`, `i`.`due_date` AS `due_date`, `i`.`total_amount` AS `total_amount`, `i`.`paid_amount` AS `paid_amount`, `i`.`balance_due` AS `balance_due`, `c`.`customerID` AS `customer_id`, coalesce(`c`.`companyname`,concat(`c`.`firstname`,' ',`c`.`lastname`)) AS `customer_name`, `j`.`jobID` AS `job_id`, `j`.`description` AS `job_description`, (to_days(curdate()) - to_days(`i`.`due_date`)) AS `days_overdue`, count(`ili`.`line_item_id`) AS `item_count` FROM (((`invoices` `i` left join `customers` `c` on((`i`.`customer_id` = `c`.`customerID`))) left join `jobs` `j` on((`i`.`job_id` = `j`.`jobID`))) left join `invoice_line_items` `ili` on((`i`.`invoice_id` = `ili`.`invoice_id`))) GROUP BY `i`.`invoice_id`, `i`.`invoice_number`, `i`.`status`, `i`.`issue_date`, `i`.`due_date`, `i`.`total_amount`, `i`.`paid_amount`, `i`.`balance_due`, `c`.`customerID`, `c`.`companyname`, `c`.`firstname`, `c`.`lastname`, `j`.`jobID`, `j`.`description` ;

-- --------------------------------------------------------

--
-- Structure for view `vw_package_devices_detail`
--
DROP TABLE IF EXISTS `vw_package_devices_detail`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `vw_package_devices_detail`  AS SELECT `pd`.`packageID` AS `packageID`, `ep`.`name` AS `packageName`, `pd`.`deviceID` AS `deviceID`, `d`.`serialnumber` AS `serialNumber`, `d`.`status` AS `deviceStatus`, `p`.`name` AS `productName`, concat(`sc`.`name`,' > ',`sbc`.`name`) AS `productCategory`, `p`.`itemcostperday` AS `defaultPrice`, `pd`.`custom_price` AS `custom_price`, coalesce(`pd`.`custom_price`,`p`.`itemcostperday`) AS `effectivePrice`, `pd`.`quantity` AS `quantity`, `pd`.`is_required` AS `is_required`, `pd`.`notes` AS `notes`, `pd`.`sort_order` AS `sort_order`, (coalesce(`pd`.`custom_price`,`p`.`itemcostperday`) * `pd`.`quantity`) AS `lineTotal` FROM (((((`package_devices` `pd` join `equipment_packages` `ep` on((`pd`.`packageID` = `ep`.`packageID`))) join `devices` `d` on((`pd`.`deviceID` = `d`.`deviceID`))) left join `products` `p` on((`d`.`productID` = `p`.`productID`))) left join `subcategories` `sc` on((`p`.`subcategoryID` = `sc`.`subcategoryID`))) left join `subbiercategories` `sbc` on((`p`.`subbiercategoryID` = `sbc`.`subbiercategoryID`))) ORDER BY `pd`.`packageID` ASC, `pd`.`sort_order` ASC, `pd`.`deviceID` ASC ;

-- --------------------------------------------------------

--
-- Structure for view `vw_package_summary`
--
DROP TABLE IF EXISTS `vw_package_summary`;

CREATE ALGORITHM=UNDEFINED DEFINER=`root`@`%` SQL SECURITY DEFINER VIEW `vw_package_summary`  AS SELECT `ep`.`packageID` AS `packageID`, `ep`.`name` AS `packageName`, `ep`.`description` AS `description`, `ep`.`package_price` AS `packagePrice`, `ep`.`discount_percent` AS `discountPercent`, `ep`.`min_rental_days` AS `minRentalDays`, `ep`.`is_active` AS `isActive`, `ep`.`usage_count` AS `usageCount`, `pc`.`name` AS `categoryName`, `pc`.`color` AS `categoryColor`, count(`pd`.`deviceID`) AS `deviceCount`, sum(`pd`.`quantity`) AS `totalDevices`, sum((case when (`pd`.`is_required` = 1) then `pd`.`quantity` else 0 end)) AS `requiredDevices`, sum((case when (`pd`.`is_required` = 0) then `pd`.`quantity` else 0 end)) AS `optionalDevices`, `ep`.`created_at` AS `createdAt`, `ep`.`updated_at` AS `updatedAt` FROM ((`equipment_packages` `ep` left join `package_categories` `pc` on((`ep`.`categoryID` = `pc`.`categoryID`))) left join `package_devices` `pd` on((`ep`.`packageID` = `pd`.`packageID`))) GROUP BY `ep`.`packageID`, `ep`.`name`, `ep`.`description`, `ep`.`package_price`, `ep`.`discount_percent`, `ep`.`min_rental_days`, `ep`.`is_active`, `ep`.`usage_count`, `pc`.`name`, `pc`.`color`, `ep`.`created_at`, `ep`.`updated_at` ;

--
-- Constraints for dumped tables
--

--
-- Constraints for table `audit_log`
--
ALTER TABLE `audit_log`
  ADD CONSTRAINT `audit_log_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE SET NULL;

--
-- Constraints for table `brands`
--
ALTER TABLE `brands`
  ADD CONSTRAINT `brands_ibfk_1` FOREIGN KEY (`manufacturerID`) REFERENCES `manufacturer` (`manufacturerID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `cables`
--
ALTER TABLE `cables`
  ADD CONSTRAINT `cables_ibfk_1` FOREIGN KEY (`connector1`) REFERENCES `cable_connectors` (`cable_connectorsID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `cables_ibfk_2` FOREIGN KEY (`connector2`) REFERENCES `cable_connectors` (`cable_connectorsID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `cables_ibfk_3` FOREIGN KEY (`typ`) REFERENCES `cable_types` (`cable_typesID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `devices`
--
ALTER TABLE `devices`
  ADD CONSTRAINT `devices_ibfk_1` FOREIGN KEY (`productID`) REFERENCES `products` (`productID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `devices_ibfk_2` FOREIGN KEY (`insuranceID`) REFERENCES `insurances` (`insuranceID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `devicescases`
--
ALTER TABLE `devicescases`
  ADD CONSTRAINT `devicescases_ibfk_1` FOREIGN KEY (`caseID`) REFERENCES `cases` (`caseID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `devicescases_ibfk_2` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `devicestatushistory`
--
ALTER TABLE `devicestatushistory`
  ADD CONSTRAINT `devicestatushistory_ibfk_1` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE CASCADE ON UPDATE CASCADE;

--
-- Constraints for table `digital_signatures`
--
ALTER TABLE `digital_signatures`
  ADD CONSTRAINT `digital_signatures_ibfk_1` FOREIGN KEY (`documentID`) REFERENCES `documents` (`documentID`) ON DELETE CASCADE;

--
-- Constraints for table `documents`
--
ALTER TABLE `documents`
  ADD CONSTRAINT `documents_ibfk_1` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL,
  ADD CONSTRAINT `documents_ibfk_2` FOREIGN KEY (`parent_documentID`) REFERENCES `documents` (`documentID`) ON DELETE SET NULL;

--
-- Constraints for table `employeejob`
--
ALTER TABLE `employeejob`
  ADD CONSTRAINT `employeejob_ibfk_1` FOREIGN KEY (`employeeID`) REFERENCES `employee` (`employeeID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `employeejob_ibfk_2` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `equipment_packages`
--
ALTER TABLE `equipment_packages`
  ADD CONSTRAINT `equipment_packages_ibfk_1` FOREIGN KEY (`created_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_equipment_packages_category` FOREIGN KEY (`categoryID`) REFERENCES `package_categories` (`categoryID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `equipment_usage_logs`
--
ALTER TABLE `equipment_usage_logs`
  ADD CONSTRAINT `equipment_usage_logs_ibfk_1` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE CASCADE,
  ADD CONSTRAINT `equipment_usage_logs_ibfk_2` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE SET NULL;

--
-- Constraints for table `financial_transactions`
--
ALTER TABLE `financial_transactions`
  ADD CONSTRAINT `financial_transactions_ibfk_1` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE CASCADE,
  ADD CONSTRAINT `financial_transactions_ibfk_2` FOREIGN KEY (`customerID`) REFERENCES `customers` (`customerID`) ON DELETE CASCADE,
  ADD CONSTRAINT `financial_transactions_ibfk_3` FOREIGN KEY (`created_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL;

--
-- Constraints for table `insurances`
--
ALTER TABLE `insurances`
  ADD CONSTRAINT `insurances_ibfk_1` FOREIGN KEY (`insuranceproviderID`) REFERENCES `insuranceprovider` (`insuranceproviderID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `invoices`
--
ALTER TABLE `invoices`
  ADD CONSTRAINT `fk_invoices_customer` FOREIGN KEY (`customer_id`) REFERENCES `customers` (`customerID`) ON DELETE RESTRICT ON UPDATE CASCADE,
  ADD CONSTRAINT `fk_invoices_job` FOREIGN KEY (`job_id`) REFERENCES `jobs` (`jobID`) ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT `fk_invoices_template` FOREIGN KEY (`template_id`) REFERENCES `invoice_templates` (`template_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT `invoices_ibfk_1` FOREIGN KEY (`created_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `invoice_line_items`
--
ALTER TABLE `invoice_line_items`
  ADD CONSTRAINT `invoice_line_items_ibfk_1` FOREIGN KEY (`invoice_id`) REFERENCES `invoices` (`invoice_id`) ON DELETE CASCADE,
  ADD CONSTRAINT `invoice_line_items_ibfk_2` FOREIGN KEY (`device_id`) REFERENCES `devices` (`deviceID`) ON DELETE SET NULL ON UPDATE CASCADE,
  ADD CONSTRAINT `invoice_line_items_ibfk_3` FOREIGN KEY (`package_id`) REFERENCES `equipment_packages` (`packageID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `invoice_payments`
--
ALTER TABLE `invoice_payments`
  ADD CONSTRAINT `invoice_payments_ibfk_1` FOREIGN KEY (`invoice_id`) REFERENCES `invoices` (`invoice_id`) ON DELETE CASCADE,
  ADD CONSTRAINT `invoice_payments_ibfk_2` FOREIGN KEY (`created_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `invoice_settings`
--
ALTER TABLE `invoice_settings`
  ADD CONSTRAINT `invoice_settings_ibfk_1` FOREIGN KEY (`updated_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `invoice_templates`
--
ALTER TABLE `invoice_templates`
  ADD CONSTRAINT `fk_invoice_templates_created_by` FOREIGN KEY (`created_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL ON UPDATE CASCADE;

--
-- Constraints for table `jobdevices`
--
ALTER TABLE `jobdevices`
  ADD CONSTRAINT `jobdevices_ibfk_2` FOREIGN KEY (`jobID`) REFERENCES `jobs` (`jobID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `jobdevices_ibfk_3` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `jobs`
--
ALTER TABLE `jobs`
  ADD CONSTRAINT `jobs_ibfk_1` FOREIGN KEY (`customerID`) REFERENCES `customers` (`customerID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `jobs_ibfk_2` FOREIGN KEY (`jobcategoryID`) REFERENCES `jobCategory` (`jobcategoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `jobs_ibfk_3` FOREIGN KEY (`statusID`) REFERENCES `status` (`statusID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `jobs_ibfk_5` FOREIGN KEY (`contract_documentID`) REFERENCES `documents` (`documentID`) ON DELETE SET NULL;

--
-- Constraints for table `maintenanceLogs`
--
ALTER TABLE `maintenanceLogs`
  ADD CONSTRAINT `fk_maintenanceLogs_device` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT `maintenanceLogs_ibfk_2` FOREIGN KEY (`employeeID`) REFERENCES `employee` (`employeeID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `offline_sync_queue`
--
ALTER TABLE `offline_sync_queue`
  ADD CONSTRAINT `offline_sync_queue_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE CASCADE;

--
-- Constraints for table `package_devices`
--
ALTER TABLE `package_devices`
  ADD CONSTRAINT `fk_package_devices_device` FOREIGN KEY (`deviceID`) REFERENCES `devices` (`deviceID`) ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT `fk_package_devices_package` FOREIGN KEY (`packageID`) REFERENCES `equipment_packages` (`packageID`) ON DELETE CASCADE ON UPDATE CASCADE;

--
-- Constraints for table `products`
--
ALTER TABLE `products`
  ADD CONSTRAINT `products_ibfk_1` FOREIGN KEY (`brandID`) REFERENCES `brands` (`brandID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `products_ibfk_2` FOREIGN KEY (`categoryID`) REFERENCES `categories` (`categoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `products_ibfk_3` FOREIGN KEY (`manufacturerID`) REFERENCES `manufacturer` (`manufacturerID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `products_ibfk_4` FOREIGN KEY (`subbiercategoryID`) REFERENCES `subbiercategories` (`subbiercategoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `products_ibfk_5` FOREIGN KEY (`subcategoryID`) REFERENCES `subcategories` (`subcategoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `push_subscriptions`
--
ALTER TABLE `push_subscriptions`
  ADD CONSTRAINT `push_subscriptions_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE CASCADE;

--
-- Constraints for table `saved_searches`
--
ALTER TABLE `saved_searches`
  ADD CONSTRAINT `saved_searches_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE CASCADE;

--
-- Constraints for table `search_history`
--
ALTER TABLE `search_history`
  ADD CONSTRAINT `search_history_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE SET NULL;

--
-- Constraints for table `subbiercategories`
--
ALTER TABLE `subbiercategories`
  ADD CONSTRAINT `subbiercategories_ibfk_1` FOREIGN KEY (`subcategoryID`) REFERENCES `subcategories` (`subcategoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `subcategories`
--
ALTER TABLE `subcategories`
  ADD CONSTRAINT `subcategories_ibfk_1` FOREIGN KEY (`categoryID`) REFERENCES `categories` (`categoryID`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `user_preferences`
--
ALTER TABLE `user_preferences`
  ADD CONSTRAINT `fk_user_preferences_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`userID`) ON DELETE CASCADE ON UPDATE CASCADE;

--
-- Constraints for table `user_roles`
--
ALTER TABLE `user_roles`
  ADD CONSTRAINT `user_roles_ibfk_1` FOREIGN KEY (`userID`) REFERENCES `users` (`userID`) ON DELETE CASCADE,
  ADD CONSTRAINT `user_roles_ibfk_2` FOREIGN KEY (`roleID`) REFERENCES `roles` (`roleID`) ON DELETE CASCADE,
  ADD CONSTRAINT `user_roles_ibfk_3` FOREIGN KEY (`assigned_by`) REFERENCES `users` (`userID`) ON DELETE SET NULL;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
