# 🗄️ RentalCore Database Setup Guide

This guide provides complete instructions for setting up the RentalCore database from scratch, perfect for new users deploying the application for the first time.

## 📋 Prerequisites

### Required Software
- **MySQL 8.0+** or **MariaDB 10.6+**
- **Database Admin Tool** (phpMyAdmin, MySQL Workbench, or command line)
- **RentalCore Application** ([Docker Hub](https://hub.docker.com/r/nbt4/rentalcore) or [GitHub](https://github.com/nbt4/RentalCore))

### System Requirements
- **RAM**: Minimum 1GB for database server
- **Storage**: 10GB+ available space (for growth)
- **Network**: MySQL port 3306 accessible to application

## 🚀 Quick Setup (Recommended)

### Option 1: Automated Setup Script
```bash
# Download the setup script
wget https://github.com/nbt4/RentalCore/raw/main/database/rentalcore_setup.sql

# Create database and user
mysql -u root -p << EOF
CREATE DATABASE rentalcore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'rentalcore_user'@'%' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON rentalcore.* TO 'rentalcore_user'@'%';
FLUSH PRIVILEGES;
EOF

# Import the database schema and sample data
mysql -u rentalcore_user -p rentalcore < rentalcore_setup.sql

echo "✅ RentalCore database setup complete!"
```

### Option 2: Docker MySQL Setup
```bash
# Create a MySQL container
docker run --name rentalcore-mysql \
  -e MYSQL_ROOT_PASSWORD=root_password \
  -e MYSQL_DATABASE=rentalcore \
  -e MYSQL_USER=rentalcore_user \
  -e MYSQL_PASSWORD=user_password \
  -p 3306:3306 \
  -d mysql:8.0

# Wait for MySQL to start
sleep 30

# Import the database schema
docker exec -i rentalcore-mysql mysql -u rentalcore_user -puser_password rentalcore < database/rentalcore_setup.sql
```

## 📝 Step-by-Step Manual Setup

### Step 1: Create Database
```sql
-- Connect to MySQL as root or admin user
CREATE DATABASE rentalcore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### Step 2: Create Database User
```sql
-- Create a dedicated user for RentalCore
CREATE USER 'rentalcore_user'@'%' IDENTIFIED BY 'your_secure_password_here';

-- Grant necessary privileges
GRANT ALL PRIVILEGES ON rentalcore.* TO 'rentalcore_user'@'%';
FLUSH PRIVILEGES;
```

### Step 3: Import Database Schema
```bash
# Option A: Using MySQL command line
mysql -u rentalcore_user -p rentalcore < database/rentalcore_setup.sql

# Option B: Using phpMyAdmin
# 1. Login to phpMyAdmin
# 2. Select the 'rentalcore' database
# 3. Go to 'Import' tab
# 4. Choose 'rentalcore_setup.sql' file
# 5. Click 'Go' to execute
```

### Step 4: Verify Installation
```sql
-- Connect to the database
USE rentalcore;

-- Check if all tables were created
SHOW TABLES;

-- Verify sample data
SELECT COUNT(*) as total_customers FROM customers;
SELECT COUNT(*) as total_products FROM products;
SELECT COUNT(*) as total_devices FROM devices;
```

## 🔧 Configuration

### Update Application Configuration

#### Environment Variables (.env)
```bash
# Database Configuration
DB_HOST=your-database-host
DB_PORT=3306
DB_NAME=rentalcore
DB_USERNAME=rentalcore_user
DB_PASSWORD=your_secure_password_here
```

#### Docker Compose (docker-compose.yml)
```yaml
services:
  rentalcore:
    image: nbt4/rentalcore:latest
    environment:
      - DB_HOST=your-database-host
      - DB_PORT=3306
      - DB_NAME=rentalcore
      - DB_USERNAME=rentalcore_user
      - DB_PASSWORD=your_secure_password_here
```

## 📊 Sample Data Overview

The setup script includes realistic sample data to help you get started:

### Equipment Inventory
- **5 Categories**: Audio, Lighting, Video, Power & Cables, Staging
- **8 Products**: Speakers, microphones, LED lights, cameras, projectors
- **10 Devices**: Ready-to-rent equipment with serial numbers

### Customer Database
- **5 Sample Customers**: Mix of companies and individuals
- **Complete Profiles**: Names, emails, addresses, phone numbers

### Rental History
- **5 Sample Jobs**: Recent completed and active rentals
- **Revenue Data**: For testing analytics and reporting features
- **Equipment Assignments**: Device-to-job relationships

### User Account
- **Default Admin**: Username `admin`, Password `admin123`
- **⚠️ Important**: Change the default password immediately after first login!

## 🔐 Security Configuration

### Database Security
```sql
-- Create a read-only user for reporting (optional)
CREATE USER 'rentalcore_readonly'@'%' IDENTIFIED BY 'readonly_password';
GRANT SELECT ON rentalcore.* TO 'rentalcore_readonly'@'%';

-- Remove test data in production (optional)
DELETE FROM jobs WHERE description LIKE '%sample%';
DELETE FROM customers WHERE email LIKE '%@email.com';
```

### Production Recommendations
```sql
-- Enable SSL for database connections
-- Add to MySQL configuration (my.cnf):
[mysqld]
ssl-ca=/path/to/ca-cert.pem
ssl-cert=/path/to/server-cert.pem
ssl-key=/path/to/server-key.pem

-- Create production admin user
CREATE USER 'admin'@'your-app-server-ip' IDENTIFIED BY 'very_secure_password';
GRANT ALL PRIVILEGES ON rentalcore.* TO 'admin'@'your-app-server-ip';
```

## 🧪 Testing the Installation

### Verify Database Connection
```bash
# Test database connectivity
mysql -h your-database-host -u rentalcore_user -p -e "SELECT 'Connection successful!' as status;"

# Test application connection
docker run --rm -e DB_HOST=your-database-host \
  -e DB_USERNAME=rentalcore_user \
  -e DB_PASSWORD=your_password \
  -e DB_NAME=rentalcore \
  nbt4/rentalcore:latest -c "SELECT 1" || echo "Connection failed"
```

### Start RentalCore Application
```bash
# Using Docker Compose
docker-compose up -d

# Check application logs
docker-compose logs -f rentalcore

# Access the application
open http://localhost:8080
```

### First Login Test
1. Navigate to `http://localhost:8080`
2. Login with username: `admin`, password: `admin123`
3. **Immediately change the password** in Profile Settings
4. Verify sample data appears in Dashboard, Devices, and Customers sections

## 📈 Performance Optimization

### Database Tuning
```sql
-- Add additional indexes for large datasets
CREATE INDEX idx_jobs_customer_date ON jobs(customerID, startDate);
CREATE INDEX idx_devices_product_serial ON devices(productID, serialnumber);
CREATE INDEX idx_analytics_performance ON analytics_cache(metric_name, period_date);
```

### MySQL Configuration Recommendations
```ini
# Add to my.cnf for better performance
[mysqld]
innodb_buffer_pool_size=1G
innodb_log_file_size=256M
max_connections=200
query_cache_size=64M
tmp_table_size=64M
max_heap_table_size=64M
```

## 🔄 Data Migration

### Importing Existing Data
```sql
-- Disable foreign key checks temporarily
SET FOREIGN_KEY_CHECKS = 0;

-- Import your data
LOAD DATA INFILE '/path/to/customers.csv' 
INTO TABLE customers 
FIELDS TERMINATED BY ',' 
LINES TERMINATED BY '\n';

-- Re-enable foreign key checks
SET FOREIGN_KEY_CHECKS = 1;
```

### Backup Procedures
```bash
# Create full database backup
mysqldump -u rentalcore_user -p --single-transaction \
  --routines --triggers rentalcore > rentalcore_backup.sql

# Automated daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d)
mysqldump -u rentalcore_user -p rentalcore | gzip > "rentalcore_backup_$DATE.sql.gz"
```

## ❓ Troubleshooting

### Common Issues

#### "Access denied" Error
```bash
# Check user permissions
mysql -u root -p -e "SELECT User, Host FROM mysql.user WHERE User='rentalcore_user';"

# Reset user password
ALTER USER 'rentalcore_user'@'%' IDENTIFIED BY 'new_password';
FLUSH PRIVILEGES;
```

#### "Table doesn't exist" Error
```bash
# Verify database selection
mysql -u rentalcore_user -p -e "USE rentalcore; SHOW TABLES;"

# Re-import schema if needed
mysql -u rentalcore_user -p rentalcore < database/rentalcore_setup.sql
```

#### Connection Timeout Issues
```bash
# Check MySQL is running and accessible
telnet your-database-host 3306

# Verify firewall allows connections
# For Ubuntu/Debian:
sudo ufw allow 3306

# For CentOS/RHEL:
sudo firewall-cmd --add-port=3306/tcp --permanent
sudo firewall-cmd --reload
```

#### Character Encoding Issues
```sql
-- Verify database charset
SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME 
FROM information_schema.SCHEMATA 
WHERE SCHEMA_NAME = 'rentalcore';

-- Fix charset if needed
ALTER DATABASE rentalcore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### Getting Help

- **GitHub Issues**: [Report problems](https://github.com/nbt4/RentalCore/issues)
- **Docker Hub**: [Image documentation](https://hub.docker.com/r/nbt4/rentalcore)
- **Database Schema**: Check `database/rentalcore_setup.sql` for complete structure

## ✅ Success Checklist

After completing the setup, verify these items:

- [ ] Database `rentalcore` created with utf8mb4 charset
- [ ] User `rentalcore_user` created with proper permissions
- [ ] All 12+ tables created successfully (customers, products, devices, jobs, etc.)
- [ ] Sample data imported (5 customers, 8 products, 10 devices)
- [ ] RentalCore application connects successfully
- [ ] Login works with admin/admin123
- [ ] **Admin password changed from default**
- [ ] Sample devices appear in inventory
- [ ] Analytics dashboard shows sample revenue data

---

**🎯 Ready to Go**: Your RentalCore database is now set up and ready for professional equipment rental management!