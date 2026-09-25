CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_run_steps_updated" ON "agent_run_steps"("updated_at", "id");

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_updated" ON "agent_proposals"("updated_at", "id");

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_decisions_created" ON "agent_decisions"("created_at", "id");

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_updated" ON "agent_runs"("updated_at", "id");

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_turns_updated" ON "assistant_turns"("updated_at", "id");

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_created" ON "ai_usage_records"("created_at", "id");

--bun:split
ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_kind";

--bun:split
ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_pair";

--bun:split
ALTER TABLE "ai_usage_records" VALIDATE CONSTRAINT "ck_ai_usage_records_owner_kind";
