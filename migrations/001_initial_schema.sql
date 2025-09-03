-- JobScanner Pro - Initial Database Schema

CREATE DATABASE IF NOT EXISTS jobscanner CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE jobscanner;

-- Customers table
CREATE TABLE customers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_email (email)
);

-- Status table
CREATE TABLE statuses (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7) DEFAULT '#007bff',
    INDEX idx_name (name)
);

-- Jobs table
CREATE TABLE jobs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    status_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_date DATE,
    end_date DATE,
    revenue DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (status_id) REFERENCES statuses(id),
    INDEX idx_customer (customer_id),
    INDEX idx_status (status_id),
    INDEX idx_dates (start_date, end_date),
    INDEX idx_revenue (revenue),
    INDEX idx_title (title),
    INDEX idx_deleted_at (deleted_at)
);

-- Devices table
CREATE TABLE devices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    serial_no VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    price DECIMAL(10,2) DEFAULT 0.00,
    available BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_serial (serial_no),
    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_available (available)
);

-- Products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    price DECIMAL(10,2) DEFAULT 0.00,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_active (active)
);

-- Job-Device relationship table
CREATE TABLE job_devices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    job_id INT NOT NULL,
    device_id INT NOT NULL,
    price DECIMAL(10,2) DEFAULT 0.00,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    INDEX idx_job (job_id),
    INDEX idx_device (device_id),
    INDEX idx_assigned (assigned_at),
    INDEX idx_removed (removed_at),
    UNIQUE KEY unique_active_assignment (job_id, device_id, removed_at)
);

-- Insert default statuses
INSERT INTO statuses (name, description, color) VALUES
('Planning', 'Job is in planning phase', '#6c757d'),
('Active', 'Job is currently active', '#28a745'),
('Completed', 'Job has been completed', '#007bff'),
('Cancelled', 'Job has been cancelled', '#dc3545'),
('On Hold', 'Job is temporarily on hold', '#ffc107');

-- Sample data removed for production use
-- To add initial data, use the application interface or create separate data import scripts

-- Create views for commonly used queries
CREATE VIEW job_summary AS
SELECT 
    j.id,
    j.title,
    j.description,
    j.start_date,
    j.end_date,
    j.revenue,
    j.created_at,
    j.updated_at,
    c.name as customer_name,
    s.name as status_name,
    s.color as status_color,
    COUNT(DISTINCT jd.device_id) as device_count,
    COALESCE(SUM(jd.price), 0) as total_device_revenue
FROM jobs j
LEFT JOIN customers c ON j.customer_id = c.id
LEFT JOIN statuses s ON j.status_id = s.id
LEFT JOIN job_devices jd ON j.id = jd.job_id AND jd.removed_at IS NULL
WHERE j.deleted_at IS NULL
GROUP BY j.id, j.title, j.description, j.start_date, j.end_date, j.revenue, j.created_at, j.updated_at, c.name, s.name, s.color;

CREATE VIEW device_status AS
SELECT 
    d.id,
    d.serial_no,
    d.name,
    d.description,
    d.category,
    d.price,
    d.available,
    d.created_at,
    d.updated_at,
    CASE 
        WHEN jd.job_id IS NOT NULL THEN FALSE 
        ELSE TRUE 
    END as is_free,
    jd.job_id as current_job_id,
    j.title as current_job_title
FROM devices d
LEFT JOIN job_devices jd ON d.id = jd.device_id AND jd.removed_at IS NULL
LEFT JOIN jobs j ON jd.job_id = j.id AND j.deleted_at IS NULL;