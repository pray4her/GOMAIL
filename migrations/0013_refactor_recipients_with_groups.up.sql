-- Migration to refactor recipient handling to use a group-based model.
-- This is a destructive migration as it removes the old direct task-to-recipient mapping.

-- Step 1: Create the recipient_groups table to define static or dynamic groups.
CREATE TABLE recipient_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    group_type VARCHAR(50) NOT NULL CHECK (group_type IN ('static', 'dynamic')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Step 2: Create the recipient_group_rules table for dynamic groups.
-- These rules will be translated into SQL queries by the application.
CREATE TABLE recipient_group_rules (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES recipient_groups(id) ON DELETE CASCADE,
    -- Field can be a direct column like 'status' or a nested JSON field like 'metadata.country'
    field VARCHAR(255) NOT NULL,
    -- Operator defines the comparison logic, e.g., 'equals', 'contains', 'greater_than'
    operator VARCHAR(50) NOT NULL,
    -- Value is the operand for the comparison.
    value TEXT NOT NULL
);

-- Step 3: Create the recipient_group_members table for static groups.
-- This table links recipients to a static group.
CREATE TABLE recipient_group_members (
    group_id BIGINT NOT NULL REFERENCES recipient_groups(id) ON DELETE CASCADE,
    recipient_id BIGINT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, recipient_id)
);

-- Step 4: Modify the email_tasks table to link to a recipient_group instead of individual recipients.
ALTER TABLE email_tasks
ADD COLUMN recipient_group_id BIGINT REFERENCES recipient_groups(id);

-- Step 5: Drop the old many-to-many link table.
-- WARNING: This will delete existing recipient lists from all previous tasks.
DROP TABLE IF EXISTS email_task_recipients;

COMMENT ON TABLE recipient_groups IS 'Stores definitions for recipient groups (segments).';
COMMENT ON COLUMN recipient_groups.group_type IS 'Type of the group: ''static'' (manual list) or ''dynamic'' (rule-based).';
COMMENT ON TABLE recipient_group_rules IS 'Defines the rules for a dynamic recipient group.';
COMMENT ON COLUMN recipient_group_rules.field IS 'The recipient attribute to filter on (e.g., ''email'', ''status'', ''metadata.country'').';
COMMENT ON COLUMN recipient_group_rules.operator IS 'The comparison operator (e.g., ''equals'', ''not_equals'', ''contains'').';
COMMENT ON TABLE recipient_group_members IS 'Many-to-many link for members of a static recipient group.'; 