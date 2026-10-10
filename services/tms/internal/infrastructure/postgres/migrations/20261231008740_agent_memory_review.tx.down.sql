DROP INDEX IF EXISTS "idx_agent_memories_unreviewed_taint";

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_review";

--bun:split

ALTER TABLE "agent_memories"
    DROP COLUMN IF EXISTS "reviewed_at",
    DROP COLUMN IF EXISTS "reviewed_by_user_id";
