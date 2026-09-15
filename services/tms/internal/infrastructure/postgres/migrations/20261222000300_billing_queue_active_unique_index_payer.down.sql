DROP INDEX CONCURRENTLY IF EXISTS "uq_billing_queue_items_active_shipment_payer_bill_type";

--bun:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_billing_queue_items_active_shipment_bill_type" ON "billing_queue_items"("shipment_id", "organization_id", "business_unit_id", "bill_type")
WHERE
    "status" NOT IN ('Posted', 'Canceled');
