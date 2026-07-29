-- Removing an enum value is not supported by PostgreSQL; 'Performance' stays behind.
DROP INDEX CONCURRENTLY IF EXISTS "idx_sf_shipment_move";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_assignments_worker_history";
