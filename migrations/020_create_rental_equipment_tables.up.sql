-- Migration: Create tables for rental equipment (zugemietetes Equipment)
-- This handles equipment that is rented from external suppliers for jobs

-- Table for rental equipment items (master list of available rental items)
CREATE TABLE IF NOT EXISTS rental_equipment (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product_name VARCHAR(255) NOT NULL,
    supplier_company VARCHAR(255) NOT NULL,
    rental_price_per_day DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    description TEXT,
    category VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_product_name (product_name),
    INDEX idx_supplier_company (supplier_company),
    INDEX idx_category (category)
);

-- Bridge table for job-rental equipment assignments
CREATE TABLE IF NOT EXISTS job_rental_equipment (
    id INT AUTO_INCREMENT PRIMARY KEY,
    job_id INT NOT NULL,
    rental_equipment_id INT,
    product_name VARCHAR(255) NOT NULL, -- Allow manual entries without rental_equipment_id
    supplier_company VARCHAR(255) NOT NULL,
    rental_price_per_day DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    quantity INT DEFAULT 1,
    days_rented INT DEFAULT 1,
    total_cost DECIMAL(10, 2) GENERATED ALWAYS AS (rental_price_per_day * quantity * days_rented) STORED,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (job_id) REFERENCES jobs(jobID) ON DELETE CASCADE,
    FOREIGN KEY (rental_equipment_id) REFERENCES rental_equipment(id) ON DELETE SET NULL,

    INDEX idx_job_id (job_id),
    INDEX idx_rental_equipment_id (rental_equipment_id),
    INDEX idx_product_name (product_name),
    INDEX idx_supplier_company (supplier_company)
);

-- Insert some example rental equipment items
INSERT INTO rental_equipment (product_name, supplier_company, rental_price_per_day, description, category) VALUES
('LED Moving Head - Martin MAC Aura', 'Pro Rental GmbH', 45.00, 'Professional LED Moving Head Light', 'Light'),
('d&b V12 Line Array', 'Sound Solutions AG', 120.00, 'High-end Line Array Speaker System', 'Sound'),
('Truss System 3m Segment', 'Stage Tech Berlin', 15.00, '3 Meter Aluminum Truss Segment', 'Stage'),
('Haze Machine - Unique 2.1', 'Effect Masters', 35.00, 'Professional Haze Machine with DMX', 'Effect'),
('LED Par 64 RGBW', 'Light Rental Pro', 12.00, 'RGBW LED Par with DMX Control', 'Light'),
('Wireless Microphone Shure ULXD2', 'Audio Rent Hamburg', 25.00, 'Professional Wireless Handheld Microphone', 'Sound');