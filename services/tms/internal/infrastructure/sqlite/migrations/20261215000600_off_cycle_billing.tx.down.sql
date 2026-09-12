-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000600_off_cycle_billing.tx.down.sql
--
-- Hand-completed: the converter emits no statement for a DROP COLUMN whose table
-- it did not see created in this dialect's history. Both columns are plain TEXT
-- with no constraint or index on them, so SQLite drops them directly.

ALTER TABLE "invoices" DROP COLUMN "off_cycle_reason";

--bun:split

ALTER TABLE "invoice_runs" DROP COLUMN "off_cycle_reason";
