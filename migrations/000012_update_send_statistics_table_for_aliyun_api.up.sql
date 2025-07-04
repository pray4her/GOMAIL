-- Up migration for send_statistics table
-- Adds columns for unique tracking and removes columns that are not provided by the current Aliyun API.

-- Add new columns for unique open and click counts
ALTER TABLE send_statistics
ADD COLUMN unique_open_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN unique_click_count INTEGER NOT NULL DEFAULT 0;

-- Remove columns that cannot be populated
ALTER TABLE send_statistics
DROP COLUMN delivered_count,
DROP COLUMN failed_count,
DROP COLUMN bounce_count;
