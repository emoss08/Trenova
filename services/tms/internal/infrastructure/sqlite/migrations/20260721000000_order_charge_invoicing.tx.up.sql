-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260721000000_order_charge_invoicing.tx.up.sql

ALTER TABLE "order_charges" ADD COLUMN "invoice_id" TEXT;

--bun:split

ALTER TABLE "order_charges" ADD COLUMN "invoiced_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS idx_order_charges_invoice ON order_charges ("invoice_id")WHERE
    "invoice_id" IS NOT NULL;
