--keel:notx
-- A customer's queue items page oldest first from the index, so the page stops at its limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_shipment_customer" ON "billing_queue_items"("organization_id", "business_unit_id", "shipment_customer_id", "created_at", "id")
WHERE
    "shipment_customer_id" IS NOT NULL;

--keel:split
-- The items whose shipment delivered, or picked up, within a window are a range of the index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_shipment_delivered" ON "billing_queue_items"("organization_id", "business_unit_id", "shipment_delivered_at")
WHERE
    "shipment_delivered_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_shipment_shipped" ON "billing_queue_items"("organization_id", "business_unit_id", "shipment_shipped_at")
WHERE
    "shipment_shipped_at" IS NOT NULL;
