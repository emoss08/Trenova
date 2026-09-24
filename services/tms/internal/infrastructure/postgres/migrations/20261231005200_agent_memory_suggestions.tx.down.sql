DROP INDEX IF EXISTS "idx_agent_memories_suggestions";

--bun:split

DELETE FROM "agent_memories" WHERE "status" IN ('Suggested', 'Dismissed') OR "source" = 'Feedback';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_feedback_evidence";

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision'));

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_status";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Retired'));

--bun:split

ALTER TABLE "agent_memories" DROP COLUMN IF EXISTS "evidence";
