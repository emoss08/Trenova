-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231001800_ai_usage_error_message.tx.up.sql

ALTER TABLE "ai_usage_records" ADD COLUMN "error_message" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_failures"
    ON "ai_usage_records" ("organization_id", "business_unit_id", "created_at" DESC)WHERE "succeeded" = false;
