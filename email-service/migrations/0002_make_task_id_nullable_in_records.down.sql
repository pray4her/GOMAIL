-- First, drop the modified foreign key constraint
ALTER TABLE "email_send_records" DROP CONSTRAINT IF EXISTS "email_send_records_task_id_fkey";

-- Then, alter the column back to NOT NULL.
-- This might fail if there are records with NULL task_id. You would need to handle those first.
ALTER TABLE "email_send_records" ALTER COLUMN "task_id" SET NOT NULL;

-- Finally, add the original strict foreign key constraint back
ALTER TABLE "email_send_records"
ADD CONSTRAINT "email_send_records_task_id_fkey"
FOREIGN KEY ("task_id") REFERENCES "email_tasks"("id") ON DELETE RESTRICT; 