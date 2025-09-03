-- ============================================================================
-- RentalCore Database Setup Validation Script
-- ============================================================================
-- This script validates that the database setup was completed correctly

-- Test 1: Verify all required tables exist
SELECT 'Testing table existence...' as test_phase;

SELECT 
  COUNT(*) as total_tables,
  CASE 
    WHEN COUNT(*) >= 12 THEN 'PASS' 
    ELSE 'FAIL' 
  END as table_test_result
FROM information_schema.tables 
WHERE table_schema = DATABASE();

-- Test 2: Check table structures for key tables
SELECT 'Testing table structures...' as test_phase;

-- Verify customers table structure
SELECT 'customers' as table_name, COUNT(*) as columns 
FROM information_schema.columns 
WHERE table_schema = DATABASE() AND table_name = 'customers';

-- Verify jobs table structure  
SELECT 'jobs' as table_name, COUNT(*) as columns
FROM information_schema.columns 
WHERE table_schema = DATABASE() AND table_name = 'jobs';

-- Verify devices table structure
SELECT 'devices' as table_name, COUNT(*) as columns
FROM information_schema.columns 
WHERE table_schema = DATABASE() AND table_name = 'devices';

-- Test 3: Verify foreign key relationships
SELECT 'Testing foreign key constraints...' as test_phase;

SELECT 
  COUNT(*) as total_foreign_keys,
  CASE 
    WHEN COUNT(*) >= 5 THEN 'PASS' 
    ELSE 'FAIL' 
  END as fk_test_result
FROM information_schema.key_column_usage 
WHERE table_schema = DATABASE() 
AND referenced_table_name IS NOT NULL;

-- Test 4: Check sample data was imported
SELECT 'Testing sample data import...' as test_phase;

-- Count sample records
SELECT 'Sample Data Check' as check_name,
  (SELECT COUNT(*) FROM customers) as customers,
  (SELECT COUNT(*) FROM categories) as categories,
  (SELECT COUNT(*) FROM products) as products,  
  (SELECT COUNT(*) FROM devices) as devices,
  (SELECT COUNT(*) FROM statuses) as statuses,
  (SELECT COUNT(*) FROM jobs) as jobs,
  (SELECT COUNT(*) FROM users) as users;

-- Test 5: Verify critical indexes exist
SELECT 'Testing index creation...' as test_phase;

SELECT 
  table_name,
  index_name,
  non_unique,
  column_name
FROM information_schema.statistics 
WHERE table_schema = DATABASE()
AND table_name IN ('customers', 'jobs', 'devices', 'jobdevices')
ORDER BY table_name, index_name;

-- Test 6: Test basic queries that the application will use
SELECT 'Testing application queries...' as test_phase;

-- Test equipment query
SELECT 'Equipment Query Test' as test_type,
  d.deviceID,
  p.name as product_name,
  c.name as category_name,
  d.status
FROM devices d
JOIN products p ON d.productID = p.productID  
JOIN categories c ON p.categoryID = c.categoryID
LIMIT 3;

-- Test customer query
SELECT 'Customer Query Test' as test_type,
  customerID,
  COALESCE(companyname, CONCAT(firstname, ' ', lastname)) as customer_name,
  email
FROM customers 
LIMIT 3;

-- Test job with revenue query
SELECT 'Job Revenue Query Test' as test_type,
  j.jobID,
  c.companyname as customer,
  j.startDate,
  j.endDate,
  COALESCE(j.final_revenue, j.revenue) as revenue
FROM jobs j
JOIN customers c ON j.customerID = c.customerID
WHERE COALESCE(j.final_revenue, j.revenue) > 0
LIMIT 3;

-- Test device availability query  
SELECT 'Device Availability Test' as test_type,
  status,
  COUNT(*) as device_count
FROM devices
GROUP BY status;

-- Final validation summary
SELECT 'VALIDATION COMPLETE' as status,
  DATABASE() as database_name,
  NOW() as validation_time,
  CASE 
    WHEN (SELECT COUNT(*) FROM customers) > 0 
     AND (SELECT COUNT(*) FROM devices) > 0
     AND (SELECT COUNT(*) FROM jobs) > 0
     AND (SELECT COUNT(*) FROM users) > 0
    THEN 'DATABASE SETUP SUCCESSFUL ✓'
    ELSE 'DATABASE SETUP INCOMPLETE ✗'
  END as final_result;