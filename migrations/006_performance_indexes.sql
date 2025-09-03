-- Performance optimization indexes
-- Add indexes for commonly searched fields and join operations

-- Jobs table indexes
CREATE INDEX IF NOT EXISTS idx_jobs_description ON jobs(description);
CREATE INDEX IF NOT EXISTS idx_jobs_end_date ON jobs(endDate);
CREATE INDEX IF NOT EXISTS idx_jobs_customer_id ON jobs(customerID);
CREATE INDEX IF NOT EXISTS idx_jobs_status_id ON jobs(statusID);

-- Customers table indexes  
CREATE INDEX IF NOT EXISTS idx_customers_search ON customers(companyname, firstname, lastname);
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);

-- Devices table indexes
CREATE INDEX IF NOT EXISTS idx_devices_search ON devices(deviceID, serialnumber);
CREATE INDEX IF NOT EXISTS idx_devices_product_id ON devices(productID);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);

-- Job devices junction table
CREATE INDEX IF NOT EXISTS idx_job_devices_job_id ON jobdevices(jobID);
CREATE INDEX IF NOT EXISTS idx_job_devices_device_id ON jobdevices(deviceID);
CREATE INDEX IF NOT EXISTS idx_job_devices_composite ON jobdevices(jobID, deviceID);

-- Financial transactions indexes
CREATE INDEX IF NOT EXISTS idx_financial_transactions_status_type ON financial_transactions(status, type);
CREATE INDEX IF NOT EXISTS idx_financial_transactions_date ON financial_transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_financial_transactions_due_date ON financial_transactions(due_date);
CREATE INDEX IF NOT EXISTS idx_financial_transactions_customer ON financial_transactions(customerID);

-- Invoices table indexes
CREATE INDEX IF NOT EXISTS idx_invoices_customer_id ON invoices(customerID);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_date ON invoices(invoice_date);

-- Sessions table cleanup index
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_jobs_customer_status ON jobs(customerID, statusID);
CREATE INDEX IF NOT EXISTS idx_devices_product_status ON devices(productID, status);