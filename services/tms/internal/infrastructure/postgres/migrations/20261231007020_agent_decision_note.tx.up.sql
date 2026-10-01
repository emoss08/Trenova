ALTER TABLE "agent_decisions"
    ADD COLUMN IF NOT EXISTS "note" TEXT;

COMMENT ON COLUMN "agent_decisions"."note" IS 'What the decider told the agent with the decision, such as why they turned it down; the conversation''s follow-up turn reads it as data, never as instructions';

--bun:split

ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_note_length";

--bun:split

ALTER TABLE "agent_decisions"
    ADD CONSTRAINT "ck_agent_decisions_note_length" CHECK (
        "note" IS NULL OR char_length("note") <= 2000
    );
