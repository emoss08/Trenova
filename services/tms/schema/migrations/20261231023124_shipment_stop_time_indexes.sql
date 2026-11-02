--keel:notx
-- The delivering-today and delivery-hour quick filters read the live shipments that deliver inside
-- a window, and the deliveries-today watchlist lists them in delivery order: the index holds the
-- shipments with a delivery time, in that order, so the read is a range scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_delivery_at" ON "shipments"("organization_id", "business_unit_id", "delivery_at", "id")
WHERE
    "status" <> 'Canceled' AND "delivery_at" IS NOT NULL;

--keel:split
-- The pickup-window quick filters count the uncovered shipments whose first pickup falls inside a
-- window, and the watchlist reads the next uncovered pickup: the index orders each stage's
-- shipments by pickup time, so both are a range scan and the next pickup is its first entry.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_stage_pickup_at" ON "shipments"("organization_id", "business_unit_id", "stage_rank", "pickup_at", "id")
WHERE
    "pickup_at" IS NOT NULL;
