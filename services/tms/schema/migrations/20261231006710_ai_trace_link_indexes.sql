--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_run_steps_updated" ON "agent_run_steps"("updated_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_updated" ON "agent_proposals"("updated_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_decisions_created" ON "agent_decisions"("created_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_updated" ON "agent_runs"("updated_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_turns_updated" ON "assistant_turns"("updated_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_created" ON "ai_usage_records"("created_at", "id");

--keel:split
ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_kind";

--keel:split
ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_pair";

--keel:split
ALTER TABLE "ai_usage_records" VALIDATE CONSTRAINT "ck_ai_usage_records_owner_kind";
