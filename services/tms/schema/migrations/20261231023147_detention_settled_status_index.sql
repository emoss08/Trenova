--keel:notx
-- Detention intelligence reads two figures live that a rollup cannot add up: the distinct
-- people who waived, and whether a billed charge reached a posted or paid invoice. Both read one
-- status of a tenant's settled window; the status leads the clock so either is one range.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_detention_occurrences_settled_status_window"
    ON "detention_occurrences"("organization_id", "business_unit_id", "status", "clock_start_at")
    WHERE "is_open" IS FALSE;
