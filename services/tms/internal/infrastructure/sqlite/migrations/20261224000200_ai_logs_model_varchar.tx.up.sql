-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261224000200_ai_logs_model_varchar.tx.up.sql

ALTER TABLE "ai_logs" ADD COLUMN "provider_kind" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "ai_logs" ADD COLUMN "provider_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_logs_model" ON "ai_logs" ("model");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_logs_provider" ON "ai_logs" ("provider_id");
