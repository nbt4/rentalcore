-- Rollback migration 024: Remove pack workflow functionality

-- Drop view
DROP VIEW IF EXISTS `v_job_pack_progress`;

-- Drop product_images table
DROP TABLE IF EXISTS `product_images`;

-- Drop job_device_events table
DROP TABLE IF EXISTS `job_device_events`;

-- Remove pack workflow columns from jobdevices
ALTER TABLE `jobdevices`
DROP INDEX IF EXISTS `idx_jobdevices_job_pack`,
DROP INDEX IF EXISTS `idx_jobdevices_pack_status`,
DROP COLUMN IF EXISTS `pack_ts`,
DROP COLUMN IF EXISTS `pack_status`;