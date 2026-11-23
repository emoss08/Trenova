--keel:notx
-- The worker record page lists one driver's violations newest first since a date. The primary key
-- leads with the worker but puts the violation type before the start, so the read gathered every
-- violation the driver ever had and sorted them; this index serves the filter and the order in one
-- range scan that stops at the page limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_hos_violations_worker_start" ON "worker_hos_violations"("organization_id", "business_unit_id", "worker_id", "violation_start_at" DESC);
