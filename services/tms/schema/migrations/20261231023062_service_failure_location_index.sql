--keel:notx
-- A location's service failures are listed newest first and counted: the index gives that order
-- within the location, so the page stops at its limit and the count reads the index alone.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_sf_location_created" ON "service_failures"("organization_id", "business_unit_id", "location_id", "created_at" DESC, "id" DESC);
