-- Remove progress tracking fields from email_tasks table
ALTER TABLE email_tasks 
DROP COLUMN total_recipients,
DROP COLUMN sent_count,
DROP COLUMN failed_count; 