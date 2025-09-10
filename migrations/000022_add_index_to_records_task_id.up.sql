-- Add index to the task_id column in the email_send_records table
CREATE INDEX idx_email_send_records_task_id ON email_send_records(task_id);
