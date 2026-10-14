CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_owner" ON "ai_usage_records"("organization_id", "business_unit_id", "owner_id") WHERE "owner_id" IS NOT NULL;

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_exceptions_run" ON "agent_exceptions"("organization_id", "business_unit_id", "run_id");
