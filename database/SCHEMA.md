# 📋 RentalCore Database Schema Documentation

This document provides a complete overview of the RentalCore database schema, relationships, and data structure for developers and database administrators.

## 🏗️ Schema Overview

RentalCore uses a normalized relational database design with the following core entities:

- **Equipment Management**: Categories, Products, Devices
- **Customer Management**: Customers, Jobs, Job Devices  
- **System Management**: Users, Statuses, Analytics Cache
- **Compliance**: Document archival and audit trails

## 📊 Entity Relationship Diagram

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Categories  │────│ Products    │────│ Devices     │
│             │    │             │    │             │
│ categoryID  │    │ productID   │    │ deviceID    │
│ name        │    │ categoryID  │    │ productID   │
│ description │    │ name        │    │ serialnumber│
└─────────────┘    │ itemcostperday   │ status      │
                   └─────────────┘    └─────────────┘
                                           │
                                           │
┌─────────────┐    ┌─────────────┐    ┌──┴──────────┐
│ Customers   │────│ Jobs        │────│ JobDevices  │
│             │    │             │    │             │
│ customerID  │    │ jobID       │    │ jobID       │
│ firstname   │    │ customerID  │    │ deviceID    │
│ lastname    │    │ statusID    │    │ custom_price│
│ companyname │    │ startDate   │    │ assigned_at │
│ email       │    │ endDate     │    └─────────────┘
└─────────────┘    │ revenue     │
                   │ final_revenue│
┌─────────────┐    │             │
│ Statuses    │────│             │
│             │    └─────────────┘
│ statusID    │
│ name        │
│ description │
│ color       │
└─────────────┘
```

## 📚 Table Definitions

### Core Equipment Tables

#### `categories`
Equipment category definitions for organizational structure.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `categoryID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique category identifier |
| `name` | VARCHAR(100) | NOT NULL, UNIQUE | Category name (e.g., "Audio Equipment") |
| `description` | TEXT | NULL | Detailed category description |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Indexes:**
- PRIMARY KEY (`categoryID`)
- UNIQUE KEY `unique_category_name` (`name`)

---

#### `products`
Product types/models within categories - templates for devices.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `productID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique product identifier |
| `name` | VARCHAR(100) | NOT NULL | Product name/model |
| `description` | TEXT | NULL | Product specifications and details |
| `categoryID` | INT | NOT NULL, FOREIGN KEY | Reference to categories table |
| `itemcostperday` | DECIMAL(10,2) | NOT NULL, DEFAULT 0.00 | Daily rental rate in EUR |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Relationships:**
- FOREIGN KEY (`categoryID`) REFERENCES `categories`(`categoryID`) ON DELETE RESTRICT

**Indexes:**
- PRIMARY KEY (`productID`)
- KEY `idx_category` (`categoryID`)

---

#### `devices`
Individual equipment items available for rental.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `deviceID` | VARCHAR(50) | PRIMARY KEY | Unique device identifier (e.g., "SPK001") |
| `productID` | INT | NOT NULL, FOREIGN KEY | Reference to products table |
| `serialnumber` | VARCHAR(100) | NULL | Manufacturer serial number |
| `status` | ENUM | NOT NULL, DEFAULT 'available' | Current device status |
| `condition_notes` | TEXT | NULL | Physical condition and maintenance notes |
| `purchase_date` | DATE | NULL | Original purchase date |
| `purchase_price` | DECIMAL(10,2) | NULL | Original purchase price |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Status Values:**
- `available` - Ready for rental
- `checked out` - Currently on a job
- `maintenance` - Under repair or maintenance
- `retired` - No longer in service

**Relationships:**
- FOREIGN KEY (`productID`) REFERENCES `products`(`productID`) ON DELETE RESTRICT

**Indexes:**
- PRIMARY KEY (`deviceID`)
- KEY `idx_product` (`productID`)
- KEY `idx_status` (`status`)
- KEY `idx_serial` (`serialnumber`)

### Customer & Job Management Tables

#### `customers`
Customer database with contact information and rental history.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `customerID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique customer identifier |
| `firstname` | VARCHAR(50) | NULL | Individual customer first name |
| `lastname` | VARCHAR(50) | NULL | Individual customer last name |
| `companyname` | VARCHAR(100) | NULL | Company/organization name |
| `email` | VARCHAR(100) | NOT NULL, UNIQUE | Primary email address |
| `phone` | VARCHAR(20) | NULL | Primary phone number |
| `address` | TEXT | NULL | Street address |
| `city` | VARCHAR(50) | NULL | City name |
| `postal_code` | VARCHAR(20) | NULL | Postal/ZIP code |
| `country` | VARCHAR(50) | DEFAULT 'Germany' | Country name |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Indexes:**
- PRIMARY KEY (`customerID`)
- UNIQUE KEY `unique_email` (`email`)
- KEY `idx_customer_search` (`lastname`, `companyname`, `email`)

---

#### `statuses`
Job status definitions with visual styling.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `statusID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique status identifier |
| `name` | VARCHAR(50) | NOT NULL, UNIQUE | Status name (e.g., "Active", "Completed") |
| `description` | VARCHAR(255) | NULL | Status description |
| `color` | VARCHAR(7) | DEFAULT '#6B7280' | Hex color code for UI display |
| `is_active` | TINYINT(1) | DEFAULT 1 | Whether status is currently in use |

**Indexes:**
- PRIMARY KEY (`statusID`)
- UNIQUE KEY `unique_status_name` (`name`)

---

#### `jobs`
Rental jobs with customer assignment and revenue tracking.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `jobID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique job identifier |
| `customerID` | INT | NOT NULL, FOREIGN KEY | Reference to customers table |
| `statusID` | INT | NOT NULL, DEFAULT 1, FOREIGN KEY | Reference to statuses table |
| `description` | TEXT | NULL | Job description and notes |
| `startDate` | DATETIME | NOT NULL | Job/rental start date and time |
| `endDate` | DATETIME | NULL | Job/rental end date and time |
| `location` | VARCHAR(255) | NULL | Event/job location |
| `revenue` | DECIMAL(10,2) | DEFAULT 0.00 | Calculated revenue amount |
| `final_revenue` | DECIMAL(10,2) | NULL | Final invoiced amount (after adjustments) |
| `discount` | DECIMAL(5,2) | DEFAULT 0.00 | Discount amount or percentage |
| `discount_type` | ENUM | DEFAULT 'percent' | Whether discount is 'percent' or 'amount' |
| `notes` | TEXT | NULL | Internal notes and comments |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Relationships:**
- FOREIGN KEY (`customerID`) REFERENCES `customers`(`customerID`) ON DELETE RESTRICT
- FOREIGN KEY (`statusID`) REFERENCES `statuses`(`statusID`) ON DELETE RESTRICT

**Indexes:**
- PRIMARY KEY (`jobID`)
- KEY `idx_customer` (`customerID`)
- KEY `idx_status` (`statusID`)
- KEY `idx_dates` (`startDate`, `endDate`)
- KEY `idx_jobs_revenue_period` (`endDate`, `final_revenue`, `revenue`)

---

#### `jobdevices`
Many-to-many relationship between jobs and devices with rental specifics.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique assignment record ID |
| `jobID` | INT | NOT NULL, FOREIGN KEY | Reference to jobs table |
| `deviceID` | VARCHAR(50) | NOT NULL, FOREIGN KEY | Reference to devices table |
| `custom_price` | DECIMAL(10,2) | NULL | Override price for this specific rental |
| `assigned_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When device was assigned to job |
| `returned_at` | TIMESTAMP | NULL | When device was returned (if applicable) |
| `condition_out` | TEXT | NULL | Device condition when rented out |
| `condition_in` | TEXT | NULL | Device condition when returned |

**Relationships:**
- FOREIGN KEY (`jobID`) REFERENCES `jobs`(`jobID`) ON DELETE CASCADE
- FOREIGN KEY (`deviceID`) REFERENCES `devices`(`deviceID`) ON DELETE RESTRICT

**Indexes:**
- PRIMARY KEY (`id`)
- UNIQUE KEY `unique_job_device` (`jobID`, `deviceID`)
- KEY `idx_device` (`deviceID`)
- KEY `idx_jobdevices_dates` (`assigned_at`, `returned_at`)

### System Management Tables

#### `users`
System users with authentication and role management.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGINT UNSIGNED | PRIMARY KEY, AUTO_INCREMENT | Unique user identifier |
| `username` | VARCHAR(50) | NOT NULL, UNIQUE | Login username |
| `email` | VARCHAR(100) | NOT NULL, UNIQUE | Email address |
| `password_hash` | VARCHAR(255) | NOT NULL | Hashed password (bcrypt) |
| `firstname` | VARCHAR(50) | NULL | User first name |
| `lastname` | VARCHAR(50) | NULL | User last name |
| `role` | ENUM | NOT NULL, DEFAULT 'user' | User role level |
| `is_active` | TINYINT(1) | NOT NULL, DEFAULT 1 | Whether account is active |
| `last_login` | TIMESTAMP | NULL | Last successful login timestamp |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Account creation timestamp |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Last modification timestamp |

**Role Values:**
- `admin` - Full system access and management
- `manager` - Business operations and reporting
- `user` - Basic rental operations

**Indexes:**
- PRIMARY KEY (`id`)
- UNIQUE KEY `unique_username` (`username`)
- UNIQUE KEY `unique_email` (`email`)

---

#### `analytics_cache`
Performance cache for analytics calculations and reports.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `cacheID` | INT | PRIMARY KEY, AUTO_INCREMENT | Unique cache entry ID |
| `metric_name` | VARCHAR(100) | NOT NULL | Name of cached metric |
| `period_type` | ENUM | NOT NULL | Time period granularity |
| `period_date` | DATE | NOT NULL | Date for the cached period |
| `value` | DECIMAL(15,4) | NULL | Calculated metric value |
| `metadata` | JSON | NULL | Additional metric metadata |
| `updated_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP ON UPDATE | Last cache update |

**Period Types:**
- `daily` - Daily aggregated metrics
- `weekly` - Weekly aggregated metrics  
- `monthly` - Monthly aggregated metrics
- `yearly` - Annual aggregated metrics

**Indexes:**
- PRIMARY KEY (`cacheID`)
- UNIQUE KEY `unique_metric_period` (`metric_name`, `period_type`, `period_date`)
- KEY `idx_analytics_cache_lookup` (`metric_name`, `period_type`, `updated_at`)

## 🔗 Key Relationships

### Equipment Hierarchy
```
Categories (1) ──── (Many) Products (1) ──── (Many) Devices
```
- One category contains many product types
- One product type can have many individual devices
- Devices inherit daily rate from their product type

### Job Management Flow
```
Customers (1) ──── (Many) Jobs (Many) ──── (Many) Devices
                                    └─── JobDevices ───┘
```
- One customer can have many jobs
- One job can use many devices
- JobDevices table manages the many-to-many relationship

### Status Tracking
```
Statuses (1) ──── (Many) Jobs
```
- Each job has exactly one status at any time
- Statuses provide workflow management

## 📊 Data Integrity Rules

### Foreign Key Constraints
- **RESTRICT**: Prevents deletion if referenced records exist
  - Categories cannot be deleted if products exist
  - Products cannot be deleted if devices exist  
  - Customers cannot be deleted if jobs exist
  - Devices cannot be deleted if assigned to active jobs

- **CASCADE**: Automatically deletes related records
  - JobDevices are deleted when jobs are deleted

### Data Validation Rules
- Email addresses must be unique across customers
- Device IDs must follow naming convention (e.g., "SPK001", "LED002")
- Job end dates must be after start dates (application level)
- Custom prices override product daily rates when specified
- Revenue calculations use final_revenue if available, otherwise revenue

## 🚀 Performance Considerations

### Query Optimization
- **Customer Search**: Indexed on lastname, companyname, email for fast searches
- **Date Range Queries**: Indexed on startDate and endDate for analytics
- **Device Lookups**: Indexed on productID and status for inventory management
- **Analytics Cache**: Indexed for rapid metric retrieval

### Recommended Indexes for Large Datasets
```sql
-- Additional performance indexes for production
CREATE INDEX idx_jobs_customer_date ON jobs(customerID, startDate);
CREATE INDEX idx_devices_product_serial ON devices(productID, serialnumber);  
CREATE INDEX idx_jobdevices_device_job ON jobdevices(deviceID, jobID);
CREATE INDEX idx_analytics_metric_date ON analytics_cache(metric_name, period_date);
```

## 🔧 Maintenance Procedures

### Regular Maintenance
```sql
-- Clean up old analytics cache (keep 2 years)
DELETE FROM analytics_cache 
WHERE period_date < DATE_SUB(NOW(), INTERVAL 2 YEAR);

-- Update device status based on active jobs
UPDATE devices d 
SET status = CASE 
  WHEN EXISTS(
    SELECT 1 FROM jobdevices jd 
    JOIN jobs j ON jd.jobID = j.jobID 
    WHERE jd.deviceID = d.deviceID 
    AND j.statusID IN (1,2) 
    AND jd.returned_at IS NULL
  ) THEN 'checked out'
  ELSE 'available'
END;
```

### Data Archival
```sql
-- Archive completed jobs older than 3 years
CREATE TABLE jobs_archive LIKE jobs;
INSERT INTO jobs_archive 
SELECT * FROM jobs 
WHERE statusID IN (3,4) 
AND endDate < DATE_SUB(NOW(), INTERVAL 3 YEAR);
```

---

**📋 Schema Summary**: The RentalCore database uses a normalized design optimized for equipment rental operations, with strong referential integrity and performance indexes for fast queries and analytics.