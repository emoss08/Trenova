-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite has no
-- ALTER TABLE ADD/DROP CONSTRAINT, so the status and disposition checks are
-- enforced by the application here, and the stored open-balance column is a
-- VIRTUAL generated column. Hand-edits are preserved only if you stop
-- regenerating this file; see docs/databases.md.
-- Source: 20261223000000_invoice_void.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "voided_at" INTEGER;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "voided_by_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "void_reason" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "void_disposition" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "voided_by_adjustment_id" TEXT;

--bun:split

DROP INDEX IF EXISTS "uk_invoices_billing_queue_item";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoices_active_billing_queue_item" ON "invoices" ("billing_queue_item_id", "organization_id", "business_unit_id")WHERE
    "status" <> 'Voided';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_voided_by_adjustment" ON "invoices" ("voided_by_adjustment_id", "organization_id", "business_unit_id")WHERE
    "voided_by_adjustment_id" IS NOT NULL;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "balance_due_minor" INTEGER GENERATED ALWAYS AS (CASE WHEN "status" = 'Posted' AND "bill_type" IN ('Invoice', 'DebitMemo') THEN MAX("total_amount_minor" - "applied_amount_minor", 0) ELSE 0 END) VIRTUAL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_balance_due" ON "invoices" ("organization_id", "business_unit_id", "balance_due_minor")WHERE
    "balance_due_minor" > 0;
