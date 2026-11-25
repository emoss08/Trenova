--keel:notx
-- The agent's shipment list keeps the shipments that have sat in their status for some days: the
-- index orders each status's shipments by when they entered it, so the read is a range scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_status_changed_at" ON "shipments"("organization_id", "business_unit_id", "status", "status_changed_at", "id");

--keel:split
-- The same list keeps the shipments with a hold in force for some days: the index holds only the
-- holds in force, by when they started, so the shipments held since a day are a range scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_holds_active_started" ON "shipment_holds"("organization_id", "business_unit_id", "started_at") INCLUDE ("shipment_id")
WHERE
    "released_at" IS NULL;
