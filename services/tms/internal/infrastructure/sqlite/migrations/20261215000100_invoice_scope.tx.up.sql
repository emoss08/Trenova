-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000100_invoice_scope.tx.up.sql
--
-- Hand-completed: the converter drops the backfill UPDATEs and cannot express
-- ALTER COLUMN ... SET NOT NULL, so the column carries its default up front and
-- the backfills are rewritten as correlated subqueries.
ALTER TABLE "invoices" ADD COLUMN "scope" TEXT NOT NULL DEFAULT 'Shipment';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "period_start" INTEGER;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "period_end" INTEGER;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "shipment_count" INTEGER NOT NULL DEFAULT 0;

--bun:split

UPDATE
    "invoices"
SET
    "scope" = CASE WHEN "is_adjustment_artifact" = 1
        OR "correction_group_id" IS NOT NULL THEN
        'Adjustment'
    WHEN "order_id" IS NOT NULL
        AND "shipment_id" IS NULL THEN
        'Order'
    WHEN "shipment_id" IS NOT NULL THEN
        'Shipment'
    ELSE
        'Adjustment'
    END;

--bun:split

UPDATE
    "invoices"
SET
    "shipment_count" = COALESCE((
        SELECT
            COUNT(DISTINCT l."shipment_id")
        FROM "invoice_lines" l
        WHERE
            l."invoice_id" = "invoices"."id"
            AND l."organization_id" = "invoices"."organization_id"
            AND l."business_unit_id" = "invoices"."business_unit_id"
            AND l."shipment_id" IS NOT NULL), 0);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_scope ON invoices ("organization_id", "business_unit_id", "scope");
