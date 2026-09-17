-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261229000300_ai_provider_last_test.tx.up.sql

ALTER TABLE "ai_providers" ADD COLUMN "last_test" TEXT;
