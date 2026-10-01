ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision', 'Feedback', 'Conversation'));

COMMENT ON COLUMN "agent_memories"."source" IS 'Who recorded it: a person, an agent through its remember tool, a decision on a proposal, ratings people gave an agent''s output, or a conversation that was summarized';

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_conversation_evidence";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_conversation_evidence" CHECK ("source" <> 'Conversation' OR ("evidence" ? 'threadId' AND "evidence" ? 'summaryId'));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_memories_conversation_suggestions"
    ON "agent_memories" ("organization_id", "business_unit_id", "agent_definition_id", "status")
    WHERE "source" = 'Conversation';
