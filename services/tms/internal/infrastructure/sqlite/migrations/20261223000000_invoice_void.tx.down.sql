-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed. SQLite cannot drop
-- a generated column or re-add a constraint, so only the indexes are undone.
-- Hand-edits are preserved only if you stop regenerating this file; see
-- docs/databases.md.
-- Source: 20261223000000_invoice_void.tx.down.sql

DROP INDEX IF EXISTS "idx_invoices_balance_due";

--bun:split

DROP INDEX IF EXISTS "idx_invoices_voided_by_adjustment";

--bun:split

DROP INDEX IF EXISTS "uq_invoices_active_billing_queue_item";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uk_invoices_billing_queue_item" ON "invoices" ("billing_queue_item_id", "organization_id", "business_unit_id");
