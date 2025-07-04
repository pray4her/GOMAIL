-- Down migration for send_statistics table
-- Reverts the changes made in the corresponding 'up' migration.

-- Add back the columns that were removed
ALTER TABLE send_statistics
ADD COLUMN delivered_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN failed_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN bounce_count INTEGER NOT NULL DEFAULT 0;

-- Remove the columns that were added
ALTER TABLE send_statistics
DROP COLUMN unique_open_count,
DROP COLUMN unique_click_count;
