-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006520_ai_usage_feature_subject.tx.up.sql

ALTER TABLE "ai_usage_records" ADD COLUMN "feature" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "subject_type" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "subject_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_feature_time"
    ON "ai_usage_records" ("organization_id", "business_unit_id", "feature", "created_at" DESC);
