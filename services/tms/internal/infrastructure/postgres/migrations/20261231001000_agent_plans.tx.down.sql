-- Drops plans and the columns that tie proposals to them. Proposals keep
-- their own statuses, so a rollback loses only the record of how they were
-- grouped and in what order they were meant to run.
DROP INDEX IF EXISTS "idx_agent_proposals_plan";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "plan_step";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "plan_id";

--bun:split
DROP INDEX IF EXISTS "idx_agent_plans_pending";

--bun:split
DROP INDEX IF EXISTS "idx_agent_plans_run";

--bun:split
DROP TABLE IF EXISTS "agent_plans";
