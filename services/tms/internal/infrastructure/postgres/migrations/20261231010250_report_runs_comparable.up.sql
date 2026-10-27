-- A run is compared with the person's latest earlier finished run of the same
-- report; this serves that lookup, newest first, within one requester's runs of
-- one saved definition or one canned report.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_report_runs_comparable" ON "report_runs"("organization_id", "business_unit_id", "requested_by_id", "definition_id", "canned_key", "completed_at" DESC)
WHERE
    "status" = 'succeeded';
