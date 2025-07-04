-- Down migration to revert the recipient group refactoring.
-- This attempts to restore the previous schema, but data for recipient lists in tasks will be lost.

-- Step 1: Remove the foreign key column from email_tasks.
ALTER TABLE email_tasks
DROP COLUMN IF EXISTS recipient_group_id;

-- Step 2: Drop the new tables.
DROP TABLE IF EXISTS recipient_group_members;
DROP TABLE IF EXISTS recipient_group_rules;
DROP TABLE IF EXISTS recipient_groups;

-- Step 3: Recreate the old many-to-many link table.
-- Note: The original data is not recoverable. This just restores the schema.
CREATE TABLE email_task_recipients (
    email_task_id BIGINT NOT NULL REFERENCES email_tasks(id) ON DELETE CASCADE,
    recipient_id BIGINT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
    PRIMARY KEY (email_task_id, recipient_id)
);

COMMENT ON TABLE email_task_recipients IS 'Re-created many-to-many link between email tasks and recipients.'; 