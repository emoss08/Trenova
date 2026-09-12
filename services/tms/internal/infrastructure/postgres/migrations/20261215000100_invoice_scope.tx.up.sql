--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- What an invoice covers was being inferred from which foreign keys happened to
-- be null: order_id set meant grouped, shipment_id set meant single. That
-- inference is already wrong in two places (reconciliation and rerate both drop
-- the order-level lines of anything that is not a single shipment) and has no
-- answer at all for an invoice covering a customer's period across many orders.
-- Scope states it instead.
CREATE TYPE "invoice_scope_enum" AS ENUM(
    'Shipment',
    'Order',
    'Consolidated',
    'Adjustment'
);

--bun:split
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "scope" invoice_scope_enum,
    ADD COLUMN IF NOT EXISTS "period_start" bigint,
    ADD COLUMN IF NOT EXISTS "period_end" bigint,
    ADD COLUMN IF NOT EXISTS "shipment_count" integer NOT NULL DEFAULT 0;

--bun:split
-- An adjustment artifact is classified first: it carries whichever of the source
-- invoice's keys it copied, so shape alone cannot tell it apart.
UPDATE
    "invoices"
SET
    "scope" = CASE WHEN "is_adjustment_artifact" = TRUE
        OR "correction_group_id" IS NOT NULL THEN
        'Adjustment'
    WHEN "order_id" IS NOT NULL
        AND "shipment_id" IS NULL THEN
        'Order'
    WHEN "shipment_id" IS NOT NULL THEN
        'Shipment'
    ELSE
        'Adjustment'
    END::invoice_scope_enum
WHERE
    "scope" IS NULL;

--bun:split
UPDATE
    "invoices" i
SET
    "shipment_count" = COALESCE((
        SELECT
            COUNT(DISTINCT l."shipment_id")
        FROM "invoice_lines" l
        WHERE
            l."invoice_id" = i."id"
            AND l."organization_id" = i."organization_id"
            AND l."business_unit_id" = i."business_unit_id"
            AND l."shipment_id" IS NOT NULL), 0);

--bun:split
ALTER TABLE "invoices"
    ALTER COLUMN "scope" SET NOT NULL,
    ALTER COLUMN "scope" SET DEFAULT 'Shipment';

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_scope ON invoices("organization_id", "business_unit_id", "scope");
