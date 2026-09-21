ALTER TABLE "agent_controls"
    DROP CONSTRAINT IF EXISTS "ck_agent_controls_briefing_hour";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "briefing_hour_local",
    DROP COLUMN IF EXISTS "briefing_enabled";

--bun:split
DROP INDEX IF EXISTS "idx_assistant_briefings_recent";

--bun:split
DROP INDEX IF EXISTS "uq_assistant_briefings_day";

--bun:split
DROP TABLE IF EXISTS "assistant_briefings";
