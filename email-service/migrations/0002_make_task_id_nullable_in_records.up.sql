-- First, drop the existing foreign key constraint
ALTER TABLE "email_send_records" DROP CONSTRAINT IF EXISTS "email_send_records_task_id_fkey";

-- Then, alter the column to allow NULL values
ALTER TABLE "email_send_records" ALTER COLUMN "task_id" DROP NOT NULL;

-- Finally, add the foreign key constraint back, but it will now work with nullable columns.
-- A foreign key constraint on a nullable column allows NULLs, but any non-NULL value must exist in the referenced table.
ALTER TABLE "email_send_records"
ADD CONSTRAINT "email_send_records_task_id_fkey"
FOREIGN KEY ("task_id") REFERENCES "email_tasks"("id") ON DELETE SET NULL; 