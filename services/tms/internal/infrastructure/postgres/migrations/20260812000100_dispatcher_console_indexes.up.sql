-- Dispatcher console support. Enum values and concurrent index builds cannot run inside
-- the migration transaction, so they live here rather than in the .tx migration that
-- adds the dispatch_controls columns.
ALTER TYPE "agent_type_enum" ADD VALUE IF NOT EXISTS 'DispatchAssignment';

--bun:split
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'ShipmentMove';

--bun:split
-- The board scans open moves for a tenant on every render. The sort key is a lateral
-- stop window that cannot be indexed, so the index only narrows the scan by tenant and
-- status.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_console_coverage" ON "shipment_moves"("organization_id", "business_unit_id", "status");

--bun:split
-- The stop-edge and move-window laterals probe stops by move within the tenant and walk
-- them in sequence order; the INCLUDE columns let the window aggregates stay index-only.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_move_window" ON "stops"("organization_id", "business_unit_id", "shipment_move_id", "sequence") INCLUDE ("status", "scheduled_window_start", "scheduled_window_end");

--bun:split
-- Approved time off is checked for every candidate on every board render.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_pto_approved_window" ON "worker_pto"("organization_id", "business_unit_id", "worker_id", "start_date", "end_date")
WHERE
    "status" = 'Approved';

--bun:split
-- The capacity rail deliberately includes drivers flagged unavailable for dispatch, so
-- the partial predicate only covers the columns every console query filters on.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_workers_dispatch_capacity" ON "workers"("organization_id", "business_unit_id", "fleet_code_id")
WHERE
    "status" = 'Active'
    AND "can_be_assigned" = TRUE;

--bun:split
-- Open-assignment counts, commitments, workload, and lane experience all probe
-- assignments by primary worker within the tenant, always excluding archived rows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_primary_worker_active" ON "assignments"("primary_worker_id", "organization_id", "business_unit_id")
WHERE
    "archived_at" IS NULL;
