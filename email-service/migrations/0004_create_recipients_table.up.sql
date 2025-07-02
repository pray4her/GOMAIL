CREATE TABLE "recipients" (
    "id" BIGSERIAL PRIMARY KEY,
    "email" VARCHAR(255) UNIQUE NOT NULL,
    "first_name" VARCHAR(100),
    "last_name" VARCHAR(100),
    "status" VARCHAR(50) NOT NULL DEFAULT 'subscribed',
    "metadata" JSONB,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT (now()),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT (now())
);

CREATE INDEX "idx_recipients_email" ON "recipients" ("email");
CREATE INDEX "idx_recipients_status" ON "recipients" ("status"); 