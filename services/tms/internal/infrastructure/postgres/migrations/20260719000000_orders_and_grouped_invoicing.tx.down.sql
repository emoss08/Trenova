--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS idx_invoice_lines_shipment;

ALTER TABLE "invoice_lines"
    DROP COLUMN IF EXISTS "shipment_id",
    DROP COLUMN IF EXISTS "shipment_pro_number",
    DROP COLUMN IF EXISTS "shipment_bol";

--bun:split
DROP INDEX IF EXISTS idx_billing_queue_items_order;

ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "fk_billing_queue_items_order";

ALTER TABLE "billing_queue_items"
    DROP COLUMN IF EXISTS "order_id";

--bun:split
DROP INDEX IF EXISTS idx_invoices_order;

ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_order";

-- Restore the single-shipment invoice invariant. Grouped invoices (shipment_id IS
-- NULL) must be removed first for the NOT NULL to reapply.
DELETE FROM "invoices"
WHERE "shipment_id" IS NULL;

ALTER TABLE "invoices"
    ALTER COLUMN "shipment_id" SET NOT NULL;

ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "order_id",
    DROP COLUMN IF EXISTS "order_number";

--bun:split
DROP INDEX IF EXISTS idx_shipments_order;

ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "fk_shipments_order";

ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "order_id";

--bun:split
DROP TABLE IF EXISTS "orders";

--bun:split
DROP TYPE IF EXISTS order_status_enum;
