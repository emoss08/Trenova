--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- An invoice already names its anchor billing-queue item, but the reverse link
-- did not exist, so settling the siblings of a grouped invoice had to be
-- inferred from the order id plus the shipments the lines happened to carry.
-- That inference has no answer for an invoice that spans several orders.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "invoice_id" varchar(100);

--bun:split
ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "fk_billing_queue_items_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- Backfill 1 — the anchor. Every invoice names its own queue item directly.
UPDATE
    "billing_queue_items" bqi
SET
    "invoice_id" = inv."id"
FROM
    "invoices" inv
WHERE
    inv."billing_queue_item_id" = bqi."id"
    AND inv."organization_id" = bqi."organization_id"
    AND inv."business_unit_id" = bqi."business_unit_id";

--bun:split
-- Backfill 2 — the siblings of a grouped invoice. A sibling shares the order,
-- bills a shipment the invoice actually carries a line for, and is not itself an
-- adjustment origin. Deliberately conservative: a row whose invoice lost its line
-- attribution stays NULL rather than being linked on a guess, which is why the
-- repository keeps an order-scoped fallback for the posting sweep.
UPDATE
    "billing_queue_items" bqi
SET
    "invoice_id" = inv."id"
FROM
    "invoices" inv
    JOIN "invoice_lines" invl ON invl."invoice_id" = inv."id"
        AND invl."organization_id" = inv."organization_id"
        AND invl."business_unit_id" = inv."business_unit_id"
WHERE
    bqi."invoice_id" IS NULL
    AND bqi."order_id" IS NOT NULL
    AND bqi."order_id" = inv."order_id"
    AND bqi."shipment_id" = invl."shipment_id"
    AND bqi."organization_id" = inv."organization_id"
    AND bqi."business_unit_id" = inv."business_unit_id"
    -- billing_queue_items.bill_type is the billing_type enum and
    -- invoices.bill_type is varchar(50), which have no equality operator
    -- between them. Compare as text, which is how the enum renders anyway.
    AND bqi."bill_type"::text = inv."bill_type"
    AND bqi."is_adjustment_origin" = FALSE;

--bun:split
CREATE INDEX IF NOT EXISTS idx_billing_queue_items_invoice ON billing_queue_items("invoice_id", "organization_id", "business_unit_id")
WHERE
    "invoice_id" IS NOT NULL;
