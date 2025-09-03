-- Critical Performance Indexes for Production
-- Execute these indexes to resolve N+1 query problems and improve performance

-- High Priority Indexes for Core Functionality
CREATE INDEX IF NOT EXISTS idx_devices_productid ON devices(productID);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_deviceid ON devices(deviceID);

-- Job-Device Relationship Optimization
CREATE INDEX IF NOT EXISTS idx_jobdevices_deviceid ON jobdevices(deviceID);
CREATE INDEX IF NOT EXISTS idx_jobdevices_jobid ON jobdevices(jobID);
CREATE INDEX IF NOT EXISTS idx_jobdevices_composite ON jobdevices(deviceID, jobID);

-- Jobs Performance Indexes
CREATE INDEX IF NOT EXISTS idx_jobs_customerid ON jobs(customerID);
CREATE INDEX IF NOT EXISTS idx_jobs_statusid ON jobs(statusID);
CREATE INDEX IF NOT EXISTS idx_jobs_dates ON jobs(startDate, endDate);
CREATE INDEX IF NOT EXISTS idx_jobs_customer_status ON jobs(customerID, statusID);

-- Invoice Performance Indexes
CREATE INDEX IF NOT EXISTS idx_invoices_customerid ON invoices(customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_dates ON invoices(issue_date, due_date);
CREATE INDEX IF NOT EXISTS idx_invoices_number ON invoices(invoice_number);

-- Customer Relationship Indexes
CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);

-- Search Optimization Indexes
CREATE INDEX IF NOT EXISTS idx_devices_search ON devices(deviceID, serialnumber);
CREATE INDEX IF NOT EXISTS idx_customers_search_company ON customers(companyname);
CREATE INDEX IF NOT EXISTS idx_customers_search_name ON customers(firstname, lastname);

-- Product and Category Indexes
CREATE INDEX IF NOT EXISTS idx_products_categoryid ON products(categoryID);
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);

-- Composite Indexes for Complex Queries
CREATE INDEX IF NOT EXISTS idx_devices_product_status ON devices(productID, status);
CREATE INDEX IF NOT EXISTS idx_jobs_status_dates ON jobs(statusID, startDate, endDate);

-- Transaction Performance
CREATE INDEX IF NOT EXISTS idx_transactions_customerid ON transactions(customerID);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transactionDate);

-- Session Management
CREATE INDEX IF NOT EXISTS idx_sessions_userid ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Company Settings
CREATE INDEX IF NOT EXISTS idx_company_settings_updated ON company_settings(updated_at);

-- Email Templates
CREATE INDEX IF NOT EXISTS idx_email_templates_type ON email_templates(template_type);
CREATE INDEX IF NOT EXISTS idx_email_templates_default ON email_templates(template_type, is_default);

-- Invoice Templates
CREATE INDEX IF NOT EXISTS idx_invoice_templates_default ON invoice_templates(is_default);
CREATE INDEX IF NOT EXISTS idx_invoice_templates_active ON invoice_templates(is_active);

-- Invoice Line Items
CREATE INDEX IF NOT EXISTS idx_invoice_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_line_items_device ON invoice_line_items(device_id);

-- Equipment Packages
CREATE INDEX IF NOT EXISTS idx_equipment_packages_status ON equipment_packages(is_active);

-- Cases
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status);
CREATE INDEX IF NOT EXISTS idx_cases_customerid ON cases(customerID);

-- Case Device Mappings
CREATE INDEX IF NOT EXISTS idx_case_device_mappings_case ON case_device_mappings(caseID);
CREATE INDEX IF NOT EXISTS idx_case_device_mappings_device ON case_device_mappings(deviceID);