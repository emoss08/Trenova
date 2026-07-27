-- Dispatcher console support. Enum values and concurrent index builds cannot run inside
-- the migration transaction, so they live here rather than in the .tx migration that
-- adds the dispatch_controls columns.
ALTER TYPE "agent_type_enum" ADD VALUE IF NOT EXISTS 'DispatchAssignment';

--bun:split
ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'ShipmentMove';

--bun:split
-- The board scans uncovered moves for a tenant on every render. Without this the scan
-- degrades to a sequential pass over every move the organization has ever created.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_console_coverage" ON "shipment_moves"("organization_id", "business_unit_id", "status", "sequence") INCLUDE ("shipment_id");

--bun:split
-- Move windows are derived from the earliest and latest stop on each move.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_move_window" ON "stops"("organization_id", "business_unit_id", "shipment_move_id", "scheduled_window_start");

--bun:split
-- Approved time off is checked for every candidate on every board render.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_pto_approved_window" ON "worker_pto"("organization_id", "business_unit_id", "worker_id", "start_date", "end_date")
WHERE
    "status" = 'Approved';

--bun:split
-- Capacity lookups narrow the roster to drivers who can actually take work.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_workers_dispatch_capacity" ON "workers"("organization_id", "business_unit_id", "fleet_code_id")
WHERE
    "status" = 'Active'
    AND "can_be_assigned" = TRUE
    AND "available_for_dispatch" = TRUE;
