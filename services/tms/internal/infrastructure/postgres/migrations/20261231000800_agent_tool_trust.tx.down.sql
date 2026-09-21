-- Drops the trust ledger and the organization switch. Tiers a tool earned stay
-- on the agent definition, since they are ordinary tool_tiers entries by now;
-- only the record of how they were earned is lost.
ALTER TABLE "agent_controls"
    DROP CONSTRAINT IF EXISTS "chk_agent_controls_promotion_threshold";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "promotion_threshold";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "earned_autonomy";

--bun:split
DROP INDEX IF EXISTS "uq_agent_tool_trust_agent_tool";

--bun:split
DROP TABLE IF EXISTS "agent_tool_trust";
