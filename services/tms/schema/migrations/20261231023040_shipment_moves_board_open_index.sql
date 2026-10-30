--keel:notx
-- The dispatch board and its summary read the open moves, each by id with its shipment and
-- status, and look up their live assignments by move id: the partial index holds only the board's
-- moves, so the summary reads them from the index without touching the heap.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_moves_board_open" ON "shipment_moves"("organization_id", "business_unit_id", "id") INCLUDE ("shipment_id", "status")
WHERE
    "status" IN ('New', 'Assigned', 'InTransit');
