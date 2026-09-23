ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "evidence" JSONB;

COMMENT ON COLUMN "agent_memories"."evidence" IS 'For a memory drawn from feedback: the ai_feedback ids it was drawn from, the pattern they share and how many people and threads rated it. Comments are quoted here as evidence, never as instructions';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_status";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Retired', 'Suggested', 'Dismissed'));

COMMENT ON COLUMN "agent_memories"."status" IS 'Active memories are read into prompts; Retired ones are kept and no longer read; Suggested ones were drawn from feedback and wait for an administrator; Dismissed ones were refused. Only Active is ever read by an agent';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision', 'Feedback'));

COMMENT ON COLUMN "agent_memories"."source" IS 'Who recorded it: a person, an agent through its remember tool, a decision on a proposal, or ratings people gave an agent''s output';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_feedback_evidence";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_feedback_evidence" CHECK ("source" <> 'Feedback' OR "evidence" IS NOT NULL);

--bun:split
-- The suggestion sweep asks, per agent, which suggestions are pending and
-- which were dismissed recently.
CREATE INDEX IF NOT EXISTS "idx_agent_memories_suggestions"
    ON "agent_memories" ("organization_id", "business_unit_id", "agent_definition_id", "status")
    WHERE "source" = 'Feedback';
