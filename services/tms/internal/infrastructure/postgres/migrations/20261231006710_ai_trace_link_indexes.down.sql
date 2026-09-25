DROP INDEX CONCURRENTLY IF EXISTS "idx_ai_usage_records_created";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_assistant_turns_updated";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_agent_runs_updated";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_agent_decisions_created";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_agent_proposals_updated";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_agent_run_steps_updated";
