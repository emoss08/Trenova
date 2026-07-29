-- Enum values and concurrent index builds cannot run inside a migration transaction,
-- so the scoring-signal support lives here rather than in a .tx migration.
ALTER TYPE "auto_assignment_strategy_enum" ADD VALUE IF NOT EXISTS 'Performance';

--bun:split
-- Driver-attributable service failures are joined to assignments through the move to
-- build each worker's on-time history.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_sf_shipment_move" ON "service_failures"("shipment_move_id", "organization_id", "business_unit_id");

--bun:split
-- Acceptance and on-time history scan assignments by worker over a created_at window;
-- the existing partial index only covers open (non-archived) rows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_worker_history" ON "assignments"("primary_worker_id", "organization_id", "business_unit_id", "created_at")
WHERE
    "primary_worker_id" IS NOT NULL;
