--keel:notx
-- Open shipments are the work under way, a small set beside a tenant's history: the ETA and
-- tracking sweeps page them by id and the Desk counters count them, so a partial index holds only
-- them. Customer status emails read live shipments most recently touched first. A BOL's
-- shipments are checked newest first; that index replaces the one on the BOL alone and still
-- leads with it for a lookup across tenants.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_open" ON "shipments"("organization_id", "business_unit_id", "id")
WHERE
    "status" IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_live_updated" ON "shipments"("organization_id", "business_unit_id", "updated_at" DESC)
WHERE
    "status" <> 'Canceled';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_bol_created" ON "shipments"("bol", "organization_id", "business_unit_id", "created_at" DESC);

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipments_bol";
