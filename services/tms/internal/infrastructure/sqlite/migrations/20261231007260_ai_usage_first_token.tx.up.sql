-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231007260_ai_usage_first_token.tx.up.sql

ALTER TABLE "ai_usage_records" ADD COLUMN "first_token_ms" INTEGER;
