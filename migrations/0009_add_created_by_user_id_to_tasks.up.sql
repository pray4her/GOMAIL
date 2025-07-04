ALTER TABLE email_tasks
ADD COLUMN created_by_user_id BIGINT;

-- We can't add a NOT NULL constraint right away on a table with existing data.
-- We will first populate it, then add the constraint.
-- For now, we'll just add the column and a foreign key.
-- Assuming no users will be deleted, otherwise we might want to set it to NULL.
ALTER TABLE email_tasks
ADD CONSTRAINT fk_email_tasks_created_by_user
FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

-- Note: In a real migration for a live system, you would first add the column,
-- then run a script to backfill the `created_by_user_id` for existing tasks
-- (e.g., assign them to a default admin user), and only then add the NOT NULL constraint.
-- For this new feature, we will assume new tasks will have this set. 