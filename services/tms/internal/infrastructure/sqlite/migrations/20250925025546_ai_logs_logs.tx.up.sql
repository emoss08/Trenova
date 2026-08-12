-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250925025546_ai_logs_logs.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_logs"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "prompt" TEXT NOT NULL,
    "response" TEXT NOT NULL,
    "model" TEXT NOT NULL,
    "operation" TEXT NOT NULL,
    "object" TEXT NOT NULL,
    "service_tier" TEXT NOT NULL,
    "prompt_tokens" INTEGER NOT NULL,
    "completion_tokens" INTEGER NOT NULL,
    "total_tokens" INTEGER NOT NULL,
    "reasoning_tokens" INTEGER NOT NULL,
    "timestamp" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_logs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_logs_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_logs_timestamp" ON "ai_logs" ("timestamp");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_logs_org_bu" ON "ai_logs" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_logs_user" ON "ai_logs" ("user_id");
