-- Enhanced Equipment Packages Migration (MySQL Compatible)
-- Add new fields for production-ready equipment packages

-- Add new columns to equipment_packages table
-- Note: Run each ALTER TABLE separately if you get syntax errors

-- Add max_rental_days column
ALTER TABLE equipment_packages ADD COLUMN max_rental_days INT NULL;

-- Add category column  
ALTER TABLE equipment_packages ADD COLUMN category VARCHAR(50) NULL;

-- Add tags column
ALTER TABLE equipment_packages ADD COLUMN tags TEXT NULL;

-- Add last_used_at column
ALTER TABLE equipment_packages ADD COLUMN last_used_at TIMESTAMP NULL;

-- Add total_revenue column
ALTER TABLE equipment_packages ADD COLUMN total_revenue DECIMAL(12,2) DEFAULT 0.00;

-- Add indexes for better performance
CREATE INDEX idx_equipment_packages_category ON equipment_packages(category);
CREATE INDEX idx_equipment_packages_active ON equipment_packages(is_active);
CREATE INDEX idx_equipment_packages_usage ON equipment_packages(usage_count);

-- Add indexes for package_devices table
CREATE INDEX idx_package_devices_package_id ON package_devices(package_id);
CREATE INDEX idx_package_devices_device_id ON package_devices(device_id);

-- Update existing package_items to be valid JSON if NULL
UPDATE equipment_packages 
SET package_items = '[]' 
WHERE package_items IS NULL OR package_items = '';

-- Sample equipment packages removed for production use
-- Create packages through the application interface as needed