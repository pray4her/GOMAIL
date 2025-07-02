-- Step 1: Add the column, but allow it to be NULL initially.
ALTER TABLE email_tasks ADD COLUMN account_sender_id BIGINT;

-- Step 2: Update existing rows with a default value.
-- This assumes at least one sender exists in the account_senders table.
-- It sets the sender_id for all tasks that don't have one to the first available sender.
UPDATE email_tasks SET account_sender_id = (SELECT id FROM account_senders ORDER BY id LIMIT 1) WHERE account_sender_id IS NULL;

-- Step 3: Now that existing rows are populated, enforce the NOT NULL constraint.
-- Note: This will fail if no senders exist in the account_senders table,
-- in which case the UPDATE above would have set the value to NULL.
ALTER TABLE email_tasks ALTER COLUMN account_sender_id SET NOT NULL;

-- Step 4: Add the foreign key constraint.
ALTER TABLE email_tasks ADD CONSTRAINT fk_email_tasks_account_senders FOREIGN KEY (account_sender_id) REFERENCES account_senders(id) ON DELETE SET NULL; 