--keel:notx
-- The shipment activity window (the board's timeline) keeps loads with a live stop due to end
-- after the window starts. That half is the selective one, since nearly every past load has a stop
-- that started before the window ends: the index lets the read find those stops by their window
-- end, so the window costs the loads in it rather than every shipment the tenant has.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_stops_live_window_end" ON "stops"("organization_id", "business_unit_id", (COALESCE("scheduled_window_end", "scheduled_window_start")), "shipment_move_id")
WHERE
    "status" <> 'Canceled';
