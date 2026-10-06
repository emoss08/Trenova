ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "saved_memories",
    DROP COLUMN IF EXISTS "used_memory_ids";

--bun:split

DROP TABLE IF EXISTS "agent_memory_preferences";

--bun:split

DROP INDEX IF EXISTS "idx_agent_memories_role";

--bun:split

DROP INDEX IF EXISTS "idx_agent_memories_owner";

--bun:split
-- A memory kept for some people has no reading under the older scopes, and
-- reading it org-wide would hand one person's notes to everyone, so it goes.
DELETE FROM "agent_memories" WHERE "scope" IN ('User', 'Role');

--bun:split

UPDATE "agent_memories" SET "status" = 'Retired' WHERE "status" = 'Paused';

--bun:split

DELETE FROM "agent_memories" WHERE "status" IN ('Suggested', 'Dismissed') AND "source" <> 'Feedback';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_status";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Retired', 'Suggested', 'Dismissed'));

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_scope";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_scope" CHECK (
        "scope" IN ('Organization', 'Agent')
        AND ("scope" = 'Organization' OR "agent_definition_id" IS NOT NULL)
    );

--bun:split

ALTER TABLE "agent_memories"
    DROP COLUMN IF EXISTS "source_thread_id",
    DROP COLUMN IF EXISTS "role_id",
    DROP COLUMN IF EXISTS "owner_user_id";
