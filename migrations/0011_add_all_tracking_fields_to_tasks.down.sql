ALTER TABLE email_tasks
DROP COLUMN open_rate,
DROP COLUMN click_rate,
DROP COLUMN unique_open_rate,
DROP COLUMN unique_click_rate;
ALTER TABLE email_tasks DROP COLUMN unique_open_count;
ALTER TABLE email_tasks DROP COLUMN unique_click_count;
ALTER TABLE email_tasks RENAME COLUMN open_count TO total_open_count;
ALTER TABLE email_tasks RENAME COLUMN click_count TO total_click_count; 