--keel:notx
-- A tractor's or trailer's move in progress is its open assignment with the lowest move id whose
-- move is in transit: the unit's open assignments are read in move order, so the indexes carry
-- the move and replace the ones on the unit alone. The rota counts open assignments created in a
-- week by driver.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_tractor_active_move" ON "assignments"("organization_id", "business_unit_id", "tractor_id", "shipment_move_id")
WHERE
    "archived_at" IS NULL AND "tractor_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_trailer_active_move" ON "assignments"("organization_id", "business_unit_id", "trailer_id", "shipment_move_id")
WHERE
    "archived_at" IS NULL AND "trailer_id" IS NOT NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_assignments_tractor_active";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_assignments_trailer_active";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_active_created" ON "assignments"("organization_id", "business_unit_id", "created_at") INCLUDE ("primary_worker_id")
WHERE
    "archived_at" IS NULL;
