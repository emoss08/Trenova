--keel:notx
-- The watchtower pages exhausted tenders oldest first: the partial index holds only those tenders
-- in that order, so the page is read in order and stops at its limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_tenders_exhausted_created" ON "tenders"("organization_id", "business_unit_id", "created_at", "id") INCLUDE ("shipment_move_id", "shipment_id")
WHERE
    "status" = 'Exhausted';

--keel:split
-- Whether a move has a newer tender than an exhausted one is a range on the move's tenders by
-- creation time and id.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_tenders_move_created" ON "tenders"("organization_id", "business_unit_id", "shipment_move_id", "created_at", "id");
