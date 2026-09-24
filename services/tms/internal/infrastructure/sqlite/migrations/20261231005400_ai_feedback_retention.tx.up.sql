-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005400_ai_feedback_retention.tx.up.sql

ALTER TABLE "data_retention" ADD COLUMN "ai_feedback_retention_period" INTEGER NOT NULL DEFAULT 730;
