DROP INDEX IF EXISTS "idx_agent_plans_script_step";

--bun:split
ALTER TABLE "agent_plans"
    DROP COLUMN IF EXISTS "script_step_key";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "script_step_key";
