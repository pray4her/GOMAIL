-- Add the old jsonb column back to the email_tasks table
ALTER TABLE "email_tasks" ADD COLUMN "recipients" JSONB;

-- Drop the join table
DROP TABLE IF EXISTS "email_task_recipients"; 