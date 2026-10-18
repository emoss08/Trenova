ALTER TABLE "carrier_settlement_lines" DROP COLUMN IF EXISTS "added_by_agent";

--bun:split
ALTER TABLE "driver_settlement_lines" DROP COLUMN IF EXISTS "added_by_agent";

--bun:split
DROP INDEX IF EXISTS "idx_agent_proposals_automatic_money";

--bun:split
ALTER TABLE "agent_proposals" DROP COLUMN IF EXISTS "money_minor";

--bun:split
ALTER TABLE "agent_controls" DROP CONSTRAINT IF EXISTS "ck_agent_controls_agent_money_bounds";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "agent_money_day_amount_minor",
    DROP COLUMN IF EXISTS "agent_money_day_calls",
    DROP COLUMN IF EXISTS "agent_money_run_amount_minor",
    DROP COLUMN IF EXISTS "agent_money_run_calls",
    DROP COLUMN IF EXISTS "agent_money_call_ceiling_minor";
