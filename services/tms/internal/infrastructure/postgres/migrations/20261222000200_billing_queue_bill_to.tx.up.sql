-- A billing queue item is one payer's claim on a shipment. Until now the payer
-- was implied by the shipment's customer; it is now recorded on the item so a
-- shipment can carry one item per payer and each lands on that payer's
-- statement. The allocated total is that payer's share of the shipment.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "bill_to_customer_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "allocated_total_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "allocated_total_amount_minor" bigint NOT NULL DEFAULT 0;

--bun:split
UPDATE
    "billing_queue_items" bqi
SET
    "bill_to_customer_id" = sp."customer_id",
    "allocated_total_amount" = COALESCE(sp."total_charge_amount", 0),
    "allocated_total_amount_minor" = ROUND(COALESCE(sp."total_charge_amount", 0) * 100)::bigint
FROM
    "shipments" sp
WHERE
    sp."id" = bqi."shipment_id"
    AND sp."organization_id" = bqi."organization_id"
    AND sp."business_unit_id" = bqi."business_unit_id"
    AND bqi."bill_to_customer_id" IS NULL;

--bun:split
UPDATE
    "billing_queue_items" bqi
SET
    "bill_to_customer_id" = inv."customer_id",
    "allocated_total_amount" = ABS(inv."total_amount"),
    "allocated_total_amount_minor" = ABS(inv."total_amount_minor")
FROM
    "invoices" inv
WHERE
    inv."id" = bqi."source_invoice_id"
    AND inv."organization_id" = bqi."organization_id"
    AND inv."business_unit_id" = bqi."business_unit_id"
    AND bqi."bill_to_customer_id" IS NULL;

--bun:split
UPDATE
    "billing_queue_items" bqi
SET
    "bill_to_customer_id" = inv."customer_id",
    "allocated_total_amount" = ABS(inv."total_amount"),
    "allocated_total_amount_minor" = ABS(inv."total_amount_minor")
FROM
    "invoices" inv
WHERE
    inv."billing_queue_item_id" = bqi."id"
    AND inv."organization_id" = bqi."organization_id"
    AND inv."business_unit_id" = bqi."business_unit_id"
    AND bqi."bill_to_customer_id" IS NULL;

--bun:split
UPDATE
    "billing_queue_items" bqi
SET
    "bill_to_customer_id" = o."customer_id"
FROM
    "orders" o
WHERE
    o."id" = bqi."order_id"
    AND o."organization_id" = bqi."organization_id"
    AND o."business_unit_id" = bqi."business_unit_id"
    AND bqi."bill_to_customer_id" IS NULL;

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT
            1
        FROM
            "billing_queue_items"
        WHERE
            "bill_to_customer_id" IS NULL) THEN
    RAISE EXCEPTION 'billing_queue_items has rows with no resolvable bill-to customer';
END IF;
END
$$;

--bun:split
ALTER TABLE "billing_queue_items"
    ALTER COLUMN "bill_to_customer_id" SET NOT NULL;

--bun:split
ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "fk_billing_queue_items_bill_to_customer" FOREIGN KEY ("bill_to_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_billing_queue_items_bill_to" ON "billing_queue_items"("bill_to_customer_id", "organization_id", "business_unit_id", "status");

--bun:split
-- A standalone memo bills a customer with no shipment or order behind it.
ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "chk_billing_queue_items_shipment_or_order";
