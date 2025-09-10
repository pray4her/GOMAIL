ALTER TABLE account_senders
ADD COLUMN reply_to_email VARCHAR(255),
ADD COLUMN metadata JSONB;
