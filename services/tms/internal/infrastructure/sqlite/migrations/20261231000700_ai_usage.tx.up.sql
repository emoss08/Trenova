-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000700_ai_usage.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_usage_records" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider_id" TEXT,
    "provider_kind" TEXT NOT NULL,
    "model" TEXT NOT NULL,
    "task" TEXT NOT NULL,
    "surface" TEXT NOT NULL,
    "user_id" TEXT,
    "agent_definition_id" TEXT,
    "thread_id" TEXT,
    "run_id" TEXT,
    "succeeded" INTEGER NOT NULL,
    "error_class" TEXT,
    "streamed" INTEGER NOT NULL DEFAULT 0,
    "latency_ms" INTEGER NOT NULL,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "reasoning_tokens" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" REAL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_usage_records_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_usage_records_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_time"
    ON "ai_usage_records" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

ALTER TABLE "ai_providers" ADD COLUMN "input_cost_per_million" REAL;

--bun:split

ALTER TABLE "ai_providers" ADD COLUMN "output_cost_per_million" REAL;

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "latency_ms" INTEGER;

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "cost_usd" REAL;
