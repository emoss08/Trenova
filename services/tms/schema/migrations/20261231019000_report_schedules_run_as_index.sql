--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_report_schedules_run_as" ON "report_schedules"("organization_id", "business_unit_id", "run_as_id");
