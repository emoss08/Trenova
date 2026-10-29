--keel:notx
-- The ETA sweep pages open shipments by id for a subset of the open statuses: the open index
-- carries the status, so the subset is filtered in the index, and it replaces the one without it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_open_status" ON "shipments"("organization_id", "business_unit_id", "id") INCLUDE ("status")
WHERE
    "status" IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted');

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipments_open";

--keel:split
-- A shipment's comments and a thread's replies are read newest or oldest first: the shipment index
-- gives that order, so no index finds a shipment's or a thread's comments without it. It replaces
-- the index on the shipment alone, and the replies index replaces the one on the parent alone.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_comments_shipment_created" ON "shipment_comments"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC, "id" DESC);

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipment_comments_shipment";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipment_comments_parent";

--keel:split
-- Watchtower reads stops not yet arrived whose window has closed, ordered by when it closed; IFTA
-- and jurisdiction mileage find the moves completed in a period through the stops that arrived or
-- departed in it and the completed moves changed in it. Each is an index over that expression or
-- that partial set.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_missed_cutoff" ON "stops"("organization_id", "business_unit_id", (COALESCE("scheduled_window_end", "scheduled_window_start")), "id") INCLUDE ("type", "shipment_move_id")
WHERE
    "actual_arrival" IS NULL AND "status" NOT IN ('Completed', 'Canceled');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_actual_time" ON "stops"("organization_id", "business_unit_id", (COALESCE("actual_departure", "actual_arrival"))) INCLUDE ("shipment_move_id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_completed_updated" ON "shipment_moves"("organization_id", "business_unit_id", "updated_at")
WHERE
    "status" = 'Completed';

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_shipment_moves_completed";

--keel:split
-- ETAs needing attention are read in shipment, move and stop order; the reads write arrived as
-- IS FALSE, so the partial index says so too, replacing the one written as NOT arrived.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stop_etas_attention_order" ON "stop_etas"("organization_id", "business_unit_id", "shipment_id", "move_sequence", "stop_sequence")
WHERE
    "verdict" IN ('AtRisk', 'Late') AND "arrived" IS FALSE;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_stop_etas_attention";
