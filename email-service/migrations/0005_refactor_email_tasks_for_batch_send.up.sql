-- Create the join table for the many-to-many relationship
CREATE TABLE "email_task_recipients" (
    "email_task_id" BIGINT NOT NULL,
    "recipient_id" BIGINT NOT NULL,
    PRIMARY KEY ("email_task_id", "recipient_id"),
    CONSTRAINT "fk_email_task_recipients_task"
        FOREIGN KEY ("email_task_id")
        REFERENCES "email_tasks"("id")
        ON DELETE CASCADE,
    CONSTRAINT "fk_email_task_recipients_recipient"
        FOREIGN KEY ("recipient_id")
        REFERENCES "recipients"("id")
        ON DELETE CASCADE
);

-- Remove the old jsonb column from the email_tasks table
ALTER TABLE "email_tasks" DROP COLUMN "recipients"; 