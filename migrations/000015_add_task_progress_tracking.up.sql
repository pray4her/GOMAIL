-- Add progress tracking fields to email_tasks table
ALTER TABLE email_tasks 
ADD COLUMN total_recipients INT DEFAULT 0,
ADD COLUMN sent_count INT DEFAULT 0,
ADD COLUMN failed_count INT DEFAULT 0; 