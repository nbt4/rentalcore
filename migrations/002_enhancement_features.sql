-- ================================================================
-- MIGRATION 002: COMPREHENSIVE ENHANCEMENT FEATURES
-- Production-ready database extensions for advanced features
-- ================================================================

-- Start transaction for atomic migration
START TRANSACTION;

-- ================================================================
-- 1. ANALYTICS & TRACKING TABLES
-- ================================================================

-- Equipment utilization tracking
CREATE TABLE equipment_usage_logs (
    logID int AUTO_INCREMENT PRIMARY KEY,
    deviceID varchar(50) NOT NULL,
    jobID int,
    action enum('assigned', 'returned', 'maintenance', 'available') NOT NULL,
    timestamp datetime NOT NULL,
    duration_hours decimal(10,2),
    revenue_generated decimal(12,2),
    notes text,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (deviceID) REFERENCES devices(deviceID) ON DELETE CASCADE,
    FOREIGN KEY (jobID) REFERENCES jobs(jobID) ON DELETE SET NULL,
    INDEX idx_device_timestamp (deviceID, timestamp),
    INDEX idx_job_action (jobID, action),
    INDEX idx_timestamp_action (timestamp, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Financial transactions
CREATE TABLE financial_transactions (
    transactionID int AUTO_INCREMENT PRIMARY KEY,
    jobID int,
    customerID int,
    type enum('rental', 'deposit', 'payment', 'refund', 'fee', 'discount') NOT NULL,
    amount decimal(12,2) NOT NULL,
    currency varchar(3) DEFAULT 'EUR',
    status enum('pending', 'completed', 'failed', 'cancelled') NOT NULL,
    payment_method varchar(50),
    transaction_date datetime NOT NULL,
    due_date date,
    reference_number varchar(100),
    notes text,
    created_by bigint UNSIGNED,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (jobID) REFERENCES jobs(jobID) ON DELETE CASCADE,
    FOREIGN KEY (customerID) REFERENCES customers(customerID) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_customer_date (customerID, transaction_date),
    INDEX idx_status_due (status, due_date),
    INDEX idx_type_date (type, transaction_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Analytics cache for performance
CREATE TABLE analytics_cache (
    cacheID int AUTO_INCREMENT PRIMARY KEY,
    metric_name varchar(100) NOT NULL,
    period_type enum('daily', 'weekly', 'monthly', 'yearly') NOT NULL,
    period_date date NOT NULL,
    value decimal(15,4),
    metadata json,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_metric (metric_name, period_type, period_date),
    INDEX idx_metric_period (metric_name, period_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 2. DOCUMENT MANAGEMENT TABLES
-- ================================================================

-- Document storage
CREATE TABLE documents (
    documentID int AUTO_INCREMENT PRIMARY KEY,
    entity_type enum('job', 'device', 'customer', 'user', 'system') NOT NULL,
    entity_id varchar(50) NOT NULL,
    filename varchar(255) NOT NULL,
    original_filename varchar(255) NOT NULL,
    file_path varchar(500) NOT NULL,
    file_size bigint NOT NULL,
    mime_type varchar(100) NOT NULL,
    document_type enum('contract', 'manual', 'photo', 'invoice', 'receipt', 'signature', 'other') NOT NULL,
    description text,
    uploaded_by bigint UNSIGNED,
    uploaded_at timestamp DEFAULT CURRENT_TIMESTAMP,
    is_public boolean DEFAULT false,
    version int DEFAULT 1,
    parent_documentID int NULL,
    checksum varchar(64),
    FOREIGN KEY (uploaded_by) REFERENCES users(userID) ON DELETE SET NULL,
    FOREIGN KEY (parent_documentID) REFERENCES documents(documentID) ON DELETE SET NULL,
    INDEX idx_entity_type (entity_type, entity_id, document_type),
    INDEX idx_uploaded_date (uploaded_at, document_type),
    INDEX idx_filename (filename)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Digital signatures
CREATE TABLE digital_signatures (
    signatureID int AUTO_INCREMENT PRIMARY KEY,
    documentID int NOT NULL,
    signer_name varchar(100) NOT NULL,
    signer_email varchar(100),
    signer_role varchar(50),
    signature_data longtext NOT NULL,
    signed_at timestamp DEFAULT CURRENT_TIMESTAMP,
    ip_address varchar(45),
    verification_code varchar(100),
    is_verified boolean DEFAULT false,
    FOREIGN KEY (documentID) REFERENCES documents(documentID) ON DELETE CASCADE,
    INDEX idx_document_signer (documentID, signer_email),
    INDEX idx_signed_date (signed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 3. ADVANCED SEARCH & SAVED FILTERS
-- ================================================================

-- Saved searches
CREATE TABLE saved_searches (
    searchID int AUTO_INCREMENT PRIMARY KEY,
    userID bigint UNSIGNED NOT NULL,
    name varchar(100) NOT NULL,
    search_type enum('global', 'jobs', 'devices', 'customers', 'cases') NOT NULL,
    filters json NOT NULL,
    is_default boolean DEFAULT false,
    is_public boolean DEFAULT false,
    usage_count int DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_used timestamp NULL,
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE CASCADE,
    INDEX idx_user_type (userID, search_type),
    INDEX idx_usage_count (usage_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Search history for analytics
CREATE TABLE search_history (
    historyID int AUTO_INCREMENT PRIMARY KEY,
    userID bigint UNSIGNED,
    search_term varchar(500),
    search_type varchar(50),
    filters json,
    results_count int,
    execution_time_ms int,
    searched_at timestamp DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_user_date (userID, searched_at),
    INDEX idx_search_type (search_type, searched_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 4. WORKFLOW & TEMPLATES
-- ================================================================

-- Job templates
CREATE TABLE job_templates (
    templateID int AUTO_INCREMENT PRIMARY KEY,
    name varchar(100) NOT NULL,
    description text,
    jobcategoryID int,
    default_duration_days int,
    equipment_list json,
    default_notes text,
    pricing_template json,
    required_documents json,
    created_by bigint UNSIGNED,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active boolean DEFAULT true,
    usage_count int DEFAULT 0,
    FOREIGN KEY (jobcategoryID) REFERENCES jobCategory(jobcategoryID) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_category_active (jobcategoryID, is_active),
    INDEX idx_usage_count (usage_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Equipment packages
CREATE TABLE equipment_packages (
    packageID int AUTO_INCREMENT PRIMARY KEY,
    name varchar(100) NOT NULL,
    description text,
    package_items json NOT NULL,
    package_price decimal(12,2),
    discount_percent decimal(5,2) DEFAULT 0.00,
    min_rental_days int DEFAULT 1,
    is_active boolean DEFAULT true,
    created_by bigint UNSIGNED,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    usage_count int DEFAULT 0,
    FOREIGN KEY (created_by) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_active_usage (is_active, usage_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 5. SECURITY & PERMISSIONS
-- ================================================================

-- Roles and permissions
CREATE TABLE roles (
    roleID int AUTO_INCREMENT PRIMARY KEY,
    name varchar(50) NOT NULL UNIQUE,
    display_name varchar(100) NOT NULL,
    description text,
    permissions json NOT NULL,
    is_system_role boolean DEFAULT false,
    is_active boolean DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_active_system (is_active, is_system_role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- User roles (many-to-many)
CREATE TABLE user_roles (
    userID bigint UNSIGNED NOT NULL,
    roleID int NOT NULL,
    assigned_at timestamp DEFAULT CURRENT_TIMESTAMP,
    assigned_by bigint UNSIGNED,
    expires_at timestamp NULL,
    is_active boolean DEFAULT true,
    PRIMARY KEY (userID, roleID),
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE CASCADE,
    FOREIGN KEY (roleID) REFERENCES roles(roleID) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_user_active (userID, is_active),
    INDEX idx_role_active (roleID, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Audit log
CREATE TABLE audit_log (
    auditID bigint AUTO_INCREMENT PRIMARY KEY,
    userID bigint UNSIGNED,
    action varchar(100) NOT NULL,
    entity_type varchar(50) NOT NULL,
    entity_id varchar(50) NOT NULL,
    old_values json,
    new_values json,
    ip_address varchar(45),
    user_agent text,
    session_id varchar(191),
    timestamp timestamp DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE SET NULL,
    INDEX idx_entity (entity_type, entity_id),
    INDEX idx_user_time (userID, timestamp),
    INDEX idx_action_time (action, timestamp),
    INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 6. MOBILE & PWA FEATURES
-- ================================================================

-- Push notification subscriptions
CREATE TABLE push_subscriptions (
    subscriptionID int AUTO_INCREMENT PRIMARY KEY,
    userID bigint UNSIGNED NOT NULL,
    endpoint text NOT NULL,
    keys_p256dh text NOT NULL,
    keys_auth text NOT NULL,
    user_agent text,
    device_type varchar(50),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    last_used timestamp DEFAULT CURRENT_TIMESTAMP,
    is_active boolean DEFAULT true,
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE CASCADE,
    INDEX idx_user_active (userID, is_active),
    INDEX idx_last_used (last_used)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Offline sync queue
CREATE TABLE offline_sync_queue (
    queueID int AUTO_INCREMENT PRIMARY KEY,
    userID bigint UNSIGNED NOT NULL,
    action enum('create', 'update', 'delete') NOT NULL,
    entity_type varchar(50) NOT NULL,
    entity_data json NOT NULL,
    timestamp timestamp DEFAULT CURRENT_TIMESTAMP,
    synced boolean DEFAULT false,
    synced_at timestamp NULL,
    retry_count int DEFAULT 0,
    error_message text NULL,
    FOREIGN KEY (userID) REFERENCES users(userID) ON DELETE CASCADE,
    INDEX idx_user_synced (userID, synced),
    INDEX idx_timestamp_synced (timestamp, synced)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ================================================================
-- 7. EXTEND EXISTING TABLES
-- ================================================================

-- Extend users table
ALTER TABLE users 
ADD COLUMN timezone varchar(50) DEFAULT 'Europe/Berlin',
ADD COLUMN language varchar(5) DEFAULT 'en',
ADD COLUMN avatar_path varchar(500),
ADD COLUMN notification_preferences json,
ADD COLUMN last_active timestamp NULL,
ADD COLUMN login_attempts int DEFAULT 0,
ADD COLUMN locked_until timestamp NULL,
ADD COLUMN two_factor_enabled boolean DEFAULT false,
ADD COLUMN two_factor_secret varchar(100);

-- Extend jobs table
ALTER TABLE jobs
ADD COLUMN templateID int NULL,
ADD COLUMN priority enum('low', 'normal', 'high', 'urgent') DEFAULT 'normal',
ADD COLUMN internal_notes text,
ADD COLUMN customer_notes text,
ADD COLUMN estimated_revenue decimal(12,2),
ADD COLUMN actual_cost decimal(12,2) DEFAULT 0.00,
ADD COLUMN profit_margin decimal(5,2),
ADD COLUMN contract_signed boolean DEFAULT false,
ADD COLUMN contract_documentID int NULL,
ADD COLUMN completion_percentage int DEFAULT 0,
ADD FOREIGN KEY (templateID) REFERENCES job_templates(templateID) ON DELETE SET NULL,
ADD FOREIGN KEY (contract_documentID) REFERENCES documents(documentID) ON DELETE SET NULL;

-- Extend devices table
ALTER TABLE devices
ADD COLUMN qr_code varchar(255) UNIQUE,
ADD COLUMN current_location varchar(100),
ADD COLUMN gps_latitude decimal(10, 8),
ADD COLUMN gps_longitude decimal(11, 8),
ADD COLUMN condition_rating decimal(3,1) DEFAULT 5.0,
ADD COLUMN usage_hours decimal(10,2) DEFAULT 0.00,
ADD COLUMN total_revenue decimal(12,2) DEFAULT 0.00,
ADD COLUMN last_maintenance_cost decimal(10,2),
ADD COLUMN notes text,
ADD COLUMN barcode varchar(255);

-- Extend customers table
ALTER TABLE customers
ADD COLUMN tax_number varchar(50),
ADD COLUMN credit_limit decimal(12,2) DEFAULT 0.00,
ADD COLUMN payment_terms int DEFAULT 30,
ADD COLUMN preferred_payment_method varchar(50),
ADD COLUMN customer_since date,
ADD COLUMN total_lifetime_value decimal(12,2) DEFAULT 0.00,
ADD COLUMN last_job_date date,
ADD COLUMN rating decimal(3,1) DEFAULT 5.0,
ADD COLUMN billing_address text,
ADD COLUMN shipping_address text;

-- ================================================================
-- 8. CREATE PERFORMANCE INDEXES
-- ================================================================

-- Analytics performance
CREATE INDEX idx_usage_logs_device_date ON equipment_usage_logs(deviceID, timestamp);
CREATE INDEX idx_transactions_customer_date ON financial_transactions(customerID, transaction_date);
CREATE INDEX idx_transactions_status ON financial_transactions(status, due_date);

-- Search performance
CREATE FULLTEXT INDEX idx_customers_search ON customers(companyname, firstname, lastname, email);
CREATE FULLTEXT INDEX idx_jobs_search ON jobs(description, internal_notes, customer_notes);

-- Document management
CREATE INDEX idx_documents_entity ON documents(entity_type, entity_id, document_type);
CREATE INDEX idx_documents_date ON documents(uploaded_at, document_type);

-- Device performance
CREATE INDEX idx_devices_location ON devices(current_location);
CREATE INDEX idx_devices_qr ON devices(qr_code);

-- ================================================================
-- 9. INSERT DEFAULT DATA
-- ================================================================

-- Insert default roles
INSERT INTO roles (name, display_name, description, permissions, is_system_role) VALUES
('super_admin', 'Super Administrator', 'Full system access', JSON_ARRAY('*'), true),
('admin', 'Administrator', 'Administrative access', JSON_ARRAY('users.manage', 'jobs.manage', 'devices.manage', 'customers.manage', 'reports.view', 'settings.manage'), true),
('manager', 'Manager', 'Management access', JSON_ARRAY('jobs.manage', 'devices.manage', 'customers.manage', 'reports.view'), true),
('operator', 'Operator', 'Basic operational access', JSON_ARRAY('jobs.view', 'jobs.create', 'devices.view', 'customers.view', 'scan.use'), true),
('viewer', 'Viewer', 'Read-only access', JSON_ARRAY('jobs.view', 'devices.view', 'customers.view'), true);

-- Insert default job template
INSERT INTO job_templates (name, description, default_duration_days, equipment_list, default_notes, created_by)
VALUES ('Standard Rental', 'Basic equipment rental template', 7, JSON_ARRAY(), 'Standard rental terms apply', 1);

-- ================================================================
-- 10. UPDATE EXISTING DATA
-- ================================================================

-- Generate QR codes for existing devices (simple format)
UPDATE devices SET qr_code = CONCAT('QR-', deviceID) WHERE qr_code IS NULL;

-- Set customer_since date for existing customers
UPDATE customers SET customer_since = DATE(created_at) WHERE customer_since IS NULL AND created_at IS NOT NULL;

-- Initialize usage hours and revenue tracking
INSERT INTO equipment_usage_logs (deviceID, action, timestamp, notes)
SELECT deviceID, 'available', NOW(), 'Initial migration record'
FROM devices 
WHERE status = 'free';

-- ================================================================
-- 11. CREATE VIEWS FOR ANALYTICS
-- ================================================================

-- Equipment utilization view
CREATE VIEW equipment_utilization AS
SELECT 
    d.deviceID,
    p.name as product_name,
    d.status,
    d.usage_hours,
    d.total_revenue,
    COALESCE(d.total_revenue / NULLIF(d.usage_hours, 0), 0) as revenue_per_hour,
    (SELECT COUNT(*) FROM equipment_usage_logs l WHERE l.deviceID = d.deviceID AND l.action = 'assigned') as times_rented,
    d.condition_rating,
    d.last_maintenance
FROM devices d
LEFT JOIN products p ON d.productID = p.productID;

-- Customer performance view
CREATE VIEW customer_performance AS
SELECT 
    c.customerID,
    c.companyname,
    c.total_lifetime_value,
    c.rating,
    c.customer_since,
    COUNT(j.jobID) as total_jobs,
    COALESCE(SUM(j.final_revenue), 0) as total_revenue,
    MAX(j.endDate) as last_job_date,
    AVG(DATEDIFF(j.endDate, j.startDate)) as avg_rental_days
FROM customers c
LEFT JOIN jobs j ON c.customerID = j.customerID
GROUP BY c.customerID;

-- Monthly revenue view
CREATE VIEW monthly_revenue AS
SELECT 
    YEAR(j.endDate) as year,
    MONTH(j.endDate) as month,
    COUNT(j.jobID) as total_jobs,
    SUM(j.final_revenue) as total_revenue,
    AVG(j.final_revenue) as avg_job_value,
    COUNT(DISTINCT j.customerID) as unique_customers
FROM jobs j
WHERE j.endDate IS NOT NULL
GROUP BY YEAR(j.endDate), MONTH(j.endDate)
ORDER BY year DESC, month DESC;

-- Commit the migration
COMMIT;

-- ================================================================
-- MIGRATION COMPLETE
-- ================================================================
-- This migration adds comprehensive features for:
-- ✅ Analytics and tracking
-- ✅ Document management  
-- ✅ Advanced search and filters
-- ✅ Workflow improvements
-- ✅ Security and permissions
-- ✅ Mobile/PWA features
-- ✅ Financial management
-- ================================================================