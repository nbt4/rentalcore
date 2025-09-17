-- Migration: Create corrected tables for rental equipment system
-- This creates tables that match the Go models exactly

-- Table for rental equipment items (master list of available rental items)
CREATE TABLE IF NOT EXISTS rental_equipment (
    equipment_id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    product_name VARCHAR(200) NOT NULL,
    supplier_name VARCHAR(100) NOT NULL,
    rental_price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    category VARCHAR(50),
    description VARCHAR(1000),
    notes VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by INT UNSIGNED,

    INDEX idx_product_name (product_name),
    INDEX idx_supplier_name (supplier_name),
    INDEX idx_category (category),
    INDEX idx_is_active (is_active)
);

-- Bridge table for job-rental equipment assignments
CREATE TABLE IF NOT EXISTS job_rental_equipment (
    job_id INT NOT NULL,
    equipment_id INT UNSIGNED NOT NULL,
    quantity INT UNSIGNED NOT NULL DEFAULT 1,
    days_used INT UNSIGNED NOT NULL DEFAULT 1,
    total_cost DECIMAL(12, 2) NOT NULL,
    notes VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (job_id, equipment_id),
    FOREIGN KEY (job_id) REFERENCES jobs(jobID) ON DELETE CASCADE,
    FOREIGN KEY (equipment_id) REFERENCES rental_equipment(equipment_id) ON DELETE CASCADE,

    INDEX idx_job_id (job_id),
    INDEX idx_equipment_id (equipment_id),
    INDEX idx_created_at (created_at)
);

-- Insert some example rental equipment items
INSERT INTO rental_equipment (product_name, supplier_name, rental_price, description, category) VALUES
('LED Moving Head - Martin MAC Aura', 'Pro Rental GmbH', 45.00, 'Professional LED Moving Head Light', 'Lighting'),
('d&b V12 Line Array', 'Sound Solutions AG', 120.00, 'High-end Line Array Speaker System', 'Audio'),
('Truss System 3m Segment', 'Stage Tech Berlin', 15.00, '3 Meter Aluminum Truss Segment', 'Stage Equipment'),
('Haze Machine - Unique 2.1', 'Effect Masters', 35.00, 'Professional Haze Machine with DMX', 'Other'),
('LED Par 64 RGBW', 'Light Rental Pro', 12.00, 'RGBW LED Par with DMX Control', 'Lighting'),
('Wireless Microphone Shure ULXD2', 'Audio Rent Hamburg', 25.00, 'Professional Wireless Handheld Microphone', 'Audio');