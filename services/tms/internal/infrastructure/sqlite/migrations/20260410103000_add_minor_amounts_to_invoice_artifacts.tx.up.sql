-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260410103000_add_minor_amounts_to_invoice_artifacts.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "subtotal_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "other_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "total_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "applied_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_adjustments" ADD COLUMN "credit_total_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_adjustments" ADD COLUMN "rebill_total_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_adjustments" ADD COLUMN "net_delta_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_adjustment_lines" ADD COLUMN "credit_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoice_adjustment_lines" ADD COLUMN "rebill_amount_minor" INTEGER NOT NULL DEFAULT 0;
