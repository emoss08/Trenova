-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000000_billing_queue_invoice_link.tx.up.sql
--
-- Hand-completed: the converter drops the backfill UPDATEs, which are rewritten
-- here as correlated subqueries. SQLite cannot add a foreign key to an existing
-- table, so the constraint is Postgres-only.
ALTER TABLE "billing_queue_items" ADD COLUMN "invoice_id" TEXT;

--bun:split

-- Backfill 1 — the anchor. Every invoice names its own queue item directly.
UPDATE
    "billing_queue_items"
SET
    "invoice_id" = (
        SELECT
            inv."id"
        FROM "invoices" inv
        WHERE
            inv."billing_queue_item_id" = "billing_queue_items"."id"
            AND inv."organization_id" = "billing_queue_items"."organization_id"
            AND inv."business_unit_id" = "billing_queue_items"."business_unit_id")
WHERE
    EXISTS (
        SELECT
            1
        FROM "invoices" inv
        WHERE
            inv."billing_queue_item_id" = "billing_queue_items"."id"
            AND inv."organization_id" = "billing_queue_items"."organization_id"
            AND inv."business_unit_id" = "billing_queue_items"."business_unit_id");

--bun:split

-- Backfill 2 — the siblings of a grouped invoice. Deliberately conservative: a
-- row whose invoice lost its line attribution stays NULL rather than being
-- linked on a guess, which is why the repository keeps an order-scoped fallback.
UPDATE
    "billing_queue_items"
SET
    "invoice_id" = (
        SELECT
            inv."id"
        FROM "invoices" inv
            JOIN "invoice_lines" invl ON invl."invoice_id" = inv."id"
                AND invl."organization_id" = inv."organization_id"
                AND invl."business_unit_id" = inv."business_unit_id"
        WHERE
            "billing_queue_items"."order_id" = inv."order_id"
            AND "billing_queue_items"."shipment_id" = invl."shipment_id"
            AND "billing_queue_items"."organization_id" = inv."organization_id"
            AND "billing_queue_items"."business_unit_id" = inv."business_unit_id"
            AND "billing_queue_items"."bill_type" = inv."bill_type"
        LIMIT 1)
WHERE
    "invoice_id" IS NULL
    AND "order_id" IS NOT NULL
    AND "is_adjustment_origin" = 0
    AND EXISTS (
        SELECT
            1
        FROM "invoices" inv
            JOIN "invoice_lines" invl ON invl."invoice_id" = inv."id"
                AND invl."organization_id" = inv."organization_id"
                AND invl."business_unit_id" = inv."business_unit_id"
        WHERE
            "billing_queue_items"."order_id" = inv."order_id"
            AND "billing_queue_items"."shipment_id" = invl."shipment_id"
            AND "billing_queue_items"."organization_id" = inv."organization_id"
            AND "billing_queue_items"."business_unit_id" = inv."business_unit_id"
            AND "billing_queue_items"."bill_type" = inv."bill_type");

--bun:split

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_invoice ON billing_queue_items ("invoice_id", "organization_id", "business_unit_id")
WHERE
    "invoice_id" IS NOT NULL;
