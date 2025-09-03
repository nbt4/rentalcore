-- Add email configuration fields to company_settings table
ALTER TABLE company_settings 
ADD COLUMN smtp_host VARCHAR(255),
ADD COLUMN smtp_port INT,
ADD COLUMN smtp_username VARCHAR(255),
ADD COLUMN smtp_password VARCHAR(255),
ADD COLUMN smtp_from_email VARCHAR(255),
ADD COLUMN smtp_from_name VARCHAR(255),
ADD COLUMN smtp_use_tls BOOLEAN DEFAULT TRUE;