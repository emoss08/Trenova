-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000400_invoice_run_group_billing_rules.tx.up.sql

ALTER TABLE "invoice_run_groups" ADD COLUMN "minimum_amount" REAL;

--bun:split

ALTER TABLE "invoice_run_groups" ADD COLUMN "auto_bill" INTEGER NOT NULL DEFAULT 0;
