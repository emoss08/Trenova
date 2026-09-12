-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000600_off_cycle_billing.tx.up.sql

ALTER TABLE "invoice_runs" ADD COLUMN "off_cycle_reason" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "off_cycle_reason" TEXT;
