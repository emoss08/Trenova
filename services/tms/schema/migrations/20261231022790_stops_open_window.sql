--keel:notx
-- Shipments running late are found through their open stops whose window has
-- passed. Open stops are the work under way, a small set beside a tenant's
-- history, so the index holds only them, by when their window opens: a stop
-- past its window's close opened before it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_open_window" ON "stops"("organization_id", "business_unit_id", "scheduled_window_start")
WHERE
    "status" IN ('New', 'InTransit');
