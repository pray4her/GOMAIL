-- 账号管理表
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    access_key_id VARCHAR(255) NOT NULL,
    access_key_secret VARCHAR(255) NOT NULL, -- In a real system, this should be encrypted
    domain VARCHAR(255) NOT NULL,
    daily_send_limit INT NOT NULL DEFAULT 5000,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- e.g., active, inactive, suspended
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 发件人管理表
CREATE TABLE senders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    contact_info VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 账号发件人关联表
CREATE TABLE account_senders (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES senders(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL UNIQUE,
    weight INT NOT NULL DEFAULT 100,
    daily_send_limit INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- e.g., active, inactive
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, sender_id)
);

-- 邮件模板表
CREATE TABLE email_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 邮件任务表
CREATE TABLE email_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL,
    template_id BIGINT REFERENCES email_templates(id),
    subject VARCHAR(255),
    body TEXT,
    recipients JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- e.g., pending, processing, completed, failed
    scheduled_at TIMESTAMPTZ,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 邮件发送记录表
CREATE TABLE email_send_records (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES email_tasks(id) ON DELETE CASCADE,
    account_sender_id BIGINT NOT NULL REFERENCES account_senders(id),
    recipient_email VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL, -- e.g., sent, failed, delivered, opened, clicked, bounced
    aliyun_task_id VARCHAR(255),
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    last_status_update_at TIMESTAMPTZ
);

-- 发送统计表
CREATE TABLE send_statistics (
    id BIGSERIAL PRIMARY KEY,
    stat_date DATE NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    account_sender_id BIGINT NOT NULL REFERENCES account_senders(id) ON DELETE CASCADE,
    sent_count INT DEFAULT 0,
    delivered_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    open_count INT DEFAULT 0,
    click_count INT DEFAULT 0,
    bounce_count INT DEFAULT 0,
    UNIQUE(stat_date, account_id, account_sender_id)
);

-- Add indexes for performance
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_account_senders_status ON account_senders(status);
CREATE INDEX idx_email_tasks_status ON email_tasks(status);
CREATE INDEX idx_email_send_records_status ON email_send_records(status);
CREATE INDEX idx_email_send_records_recipient ON email_send_records(recipient_email);
CREATE INDEX idx_email_send_records_aliyun_task_id ON email_send_records(aliyun_task_id); 