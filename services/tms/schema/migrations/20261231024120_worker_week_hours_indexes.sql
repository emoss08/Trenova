--keel:notx
-- The board's week of hours reads the shift assignments in force this week across a tenant: the
-- open ones through uq_worker_shift_assignments_open and the ones that ended through this partial
-- index, so the read is two index range scans, not every assignment the tenant ever made.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_worker_shift_assignments_ended" ON "worker_shift_assignments"("organization_id", "business_unit_id", "effective_to")
WHERE
    "effective_to" IS NOT NULL;
