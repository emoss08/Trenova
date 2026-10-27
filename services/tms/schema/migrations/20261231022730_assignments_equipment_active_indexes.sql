--keel:notx
-- A tractor's or trailer's move in progress is read for each row of the equipment lists, and
-- checked before a move starts with it: its open assignments by unit within the tenant.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_tractor_active" ON "assignments"("organization_id", "business_unit_id", "tractor_id")
WHERE
    "archived_at" IS NULL AND "tractor_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assignments_trailer_active" ON "assignments"("organization_id", "business_unit_id", "trailer_id")
WHERE
    "archived_at" IS NULL AND "trailer_id" IS NOT NULL;
