DROP INDEX IF EXISTS "idx_agent_proposals_executed_tool";

--bun:split
ALTER TABLE "agent_proposals" DROP COLUMN IF EXISTS "simulation";

--bun:split
ALTER TABLE "agent_proposals" DROP COLUMN IF EXISTS "simulated_at";

--bun:split
ALTER TABLE "agent_definitions" DROP CONSTRAINT IF EXISTS "chk_agent_definitions_budget";

--bun:split
ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "simulation_mode";

--bun:split
ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "tool_daily_limits";

--bun:split
ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "daily_run_limit";

--bun:split
ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "monthly_budget_usd";
