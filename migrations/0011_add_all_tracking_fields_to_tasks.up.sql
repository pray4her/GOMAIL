ALTER TABLE email_tasks RENAME COLUMN total_open_count TO open_count;
ALTER TABLE email_tasks RENAME COLUMN total_click_count TO click_count;
ALTER TABLE email_tasks ADD COLUMN unique_open_count INT NOT NULL DEFAULT 0;
ALTER TABLE email_tasks ADD COLUMN unique_click_count INT NOT NULL DEFAULT 0;
ALTER TABLE email_tasks
ADD COLUMN open_rate FLOAT NOT NULL DEFAULT 0.0,
ADD COLUMN click_rate FLOAT NOT NULL DEFAULT 0.0,
ADD COLUMN unique_open_rate FLOAT NOT NULL DEFAULT 0.0,
ADD COLUMN unique_click_rate FLOAT NOT NULL DEFAULT 0.0; 