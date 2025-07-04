ALTER TABLE email_tasks
ADD COLUMN aliyun_tag_name VARCHAR(60),
ADD COLUMN aliyun_tag_id VARCHAR(255),
ADD COLUMN total_open_count INT NOT NULL DEFAULT 0,
ADD COLUMN total_click_count INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN email_tasks.aliyun_tag_name IS 'Unique tag name created in Aliyun for this task (e.g., task_101).';
COMMENT ON COLUMN email_tasks.aliyun_tag_id IS 'The ID of the tag returned by Aliyun CreateTag API.';
COMMENT ON COLUMN email_tasks.total_open_count IS 'Total open count for all emails in this task, synced from Aliyun.';
COMMENT ON COLUMN email_tasks.total_click_count IS 'Total click count for all emails in this task, synced from Aliyun.'; 