ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "chk_agent_definitions_memory_token_budget";

--bun:split

ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "memory_token_budget";

--bun:split

DROP INDEX IF EXISTS "idx_agent_memories_search";

--bun:split

ALTER TABLE "agent_memories"
    DROP COLUMN IF EXISTS "search_vector";
