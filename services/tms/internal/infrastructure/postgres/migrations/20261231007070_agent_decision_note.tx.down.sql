ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_note_length";

--bun:split

ALTER TABLE "agent_decisions"
    DROP COLUMN IF EXISTS "note";
