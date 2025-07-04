ALTER TABLE email_tasks DROP CONSTRAINT IF EXISTS fk_email_tasks_account_sender;
ALTER TABLE email_tasks DROP COLUMN IF EXISTS account_sender_id;
