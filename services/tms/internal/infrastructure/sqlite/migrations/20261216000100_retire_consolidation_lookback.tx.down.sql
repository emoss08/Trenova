-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261216000100_retire_consolidation_lookback.tx.down.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "consolidation_lookback_days" INTEGER NOT NULL DEFAULT 30;
