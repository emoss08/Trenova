--keel:notx
-- A shipment's moves are read in sequence (the previous trailer on the board, the latest carrier
-- move) and checked for status beside the assignment counters, so the index carries the
-- sequence and the status. IFTA and jurisdiction mileage read every completed move's distance
-- and the time it last changed; the carrier check-call sweep pages carrier moves in transit by
-- id. Both are partial indexes over the moves they read.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_shipment_sequence" ON "shipment_moves"("organization_id", "business_unit_id", "shipment_id", "sequence") INCLUDE ("status");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_completed" ON "shipment_moves"("organization_id", "business_unit_id", "id") INCLUDE ("loaded", "distance", "distance_units", "updated_at")
WHERE
    "status" = 'Completed';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_carrier_in_transit" ON "shipment_moves"("organization_id", "business_unit_id", "id")
WHERE
    "status" = 'InTransit' AND "coverage_type" = 'carrier';
