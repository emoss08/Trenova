-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261216000100_retire_consolidation_lookback.tx.up.sql
--
-- Hand-completed: the converter emits no statement for a DROP COLUMN whose table
-- it did not see created in this dialect's history. The column is a plain
-- INTEGER with a default and no constraint or index, so SQLite drops it directly.

ALTER TABLE "customer_billing_profiles" DROP COLUMN "consolidation_lookback_days";
