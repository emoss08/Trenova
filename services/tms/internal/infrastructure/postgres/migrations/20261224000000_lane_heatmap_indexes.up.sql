-- Concurrent index builds cannot run inside the migration transaction.
-- The lane heatmap and window analytics filter a tenant's shipments by creation window
-- while excluding a status. The existing status-led index cannot range-seek a window
-- behind a `!=` predicate, so this keys on the window and carries status for the filter.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_org_bu_created_at" ON "shipments"("organization_id", "business_unit_id", "created_at") INCLUDE ("status");
