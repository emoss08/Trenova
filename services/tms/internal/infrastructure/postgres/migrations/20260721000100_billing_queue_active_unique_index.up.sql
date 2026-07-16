-- One active (not yet posted, not canceled) billing pipeline per shipment and bill
-- type. Posted items stay around for history and adjustment replacements; canceled
-- items may be re-billed. This is the concurrency backstop for the app-level
-- ExistsByShipmentAndType check (double-click / concurrent CreateFromOrder).
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_billing_queue_items_active_shipment_bill_type" ON "billing_queue_items"("shipment_id", "organization_id", "business_unit_id", "bill_type")
WHERE
    "status" NOT IN ('Posted', 'Canceled');
