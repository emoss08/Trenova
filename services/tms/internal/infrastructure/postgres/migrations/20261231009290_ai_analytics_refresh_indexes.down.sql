DROP INDEX CONCURRENTLY IF EXISTS "idx_assistant_turns_tenant_updated";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_agent_runs_tenant_updated";
