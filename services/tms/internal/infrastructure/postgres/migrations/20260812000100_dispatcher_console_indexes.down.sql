-- Enum values are intentionally not removed: PostgreSQL cannot drop a value from an
-- existing type, and leaving them in place is harmless once nothing references them.
DROP INDEX CONCURRENTLY IF EXISTS "idx_assignments_primary_worker_active";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_workers_dispatch_capacity";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_worker_pto_approved_window";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_stops_move_window";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipment_moves_console_coverage";
