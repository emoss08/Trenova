--keel:notx
-- A move's stops are read in sequence for their windows, their actual times and their type (the
-- board's origin and destination, IFTA's completion time, a shipment's last delivery), so the
-- move index carries them and replaces the one that carried the windows alone. The on-time card
-- reads completed deliveries by arrival, the detention watchlist reads stops arrived and not yet
-- departed by arrival, and the pickups card reads pickups by window: each a partial index over the
-- stops it reads.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_move_progress" ON "stops"("organization_id", "business_unit_id", "shipment_move_id", "sequence") INCLUDE ("status", "type", "scheduled_window_start", "scheduled_window_end", "actual_arrival", "actual_departure");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_stops_move_window";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_delivered_arrival" ON "stops"("organization_id", "business_unit_id", "actual_arrival")
WHERE
    "status" = 'Completed' AND "type" IN ('Delivery', 'SplitDelivery');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_arrived_open" ON "stops"("organization_id", "business_unit_id", "actual_arrival")
WHERE
    "status" <> 'Canceled' AND "actual_arrival" > 0 AND ("actual_departure" IS NULL OR "actual_departure" = 0);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_pickup_window" ON "stops"("organization_id", "business_unit_id", "scheduled_window_start")
WHERE
    "type" IN ('Pickup', 'SplitPickup') AND "status" <> 'Canceled';
