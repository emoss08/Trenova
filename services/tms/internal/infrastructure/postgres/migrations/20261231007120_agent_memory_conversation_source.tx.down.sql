DROP INDEX IF EXISTS "idx_agent_memories_conversation_suggestions";

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_conversation_evidence";

--bun:split
DELETE FROM "agent_memories" WHERE "source" = 'Conversation';

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision', 'Feedback'));
