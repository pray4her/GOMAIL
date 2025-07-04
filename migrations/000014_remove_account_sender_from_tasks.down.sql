ALTER TABLE email_tasks ADD COLUMN account_sender_id BIGINT;

UPDATE email_tasks SET account_sender_id = 1 WHERE account_sender_id IS NULL;

ALTER TABLE email_tasks ALTER COLUMN account_sender_id SET NOT NULL;

ALTER TABLE email_tasks 
ADD CONSTRAINT fk_email_tasks_account_sender 
FOREIGN KEY (account_sender_id) 
REFERENCES account_senders(id);
