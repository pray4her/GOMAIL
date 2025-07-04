ALTER TABLE email_tasks
DROP CONSTRAINT IF EXISTS fk_email_tasks_created_by_user;

ALTER TABLE email_tasks
DROP COLUMN IF EXISTS created_by_user_id; 