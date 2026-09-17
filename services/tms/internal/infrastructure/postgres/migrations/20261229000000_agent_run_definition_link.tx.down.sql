DROP INDEX IF EXISTS "idx_agent_runs_definition";

--bun:split
ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_trigger";

--bun:split
ALTER TABLE "agent_runs"
    DROP COLUMN IF EXISTS "summary",
    DROP COLUMN IF EXISTS "trigger",
    DROP COLUMN IF EXISTS "agent_definition_id";
