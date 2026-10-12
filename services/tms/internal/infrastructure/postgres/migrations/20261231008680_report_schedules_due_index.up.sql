CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_report_schedules_due" ON "report_schedules"("next_run_at") WHERE "enabled" AND "next_run_at" IS NOT NULL;
