ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "chk_billing_queue_items_shipment_or_order" CHECK ("shipment_id" IS NOT NULL OR "order_id" IS NOT NULL) NOT VALID;

--bun:split
DROP INDEX IF EXISTS "idx_billing_queue_items_bill_to";

--bun:split
ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "fk_billing_queue_items_bill_to_customer";

--bun:split
ALTER TABLE "billing_queue_items"
    DROP COLUMN IF EXISTS "bill_to_customer_id",
    DROP COLUMN IF EXISTS "allocated_total_amount",
    DROP COLUMN IF EXISTS "allocated_total_amount_minor";
