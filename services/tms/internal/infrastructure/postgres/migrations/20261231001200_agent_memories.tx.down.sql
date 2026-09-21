DROP INDEX IF EXISTS "idx_agent_memories_created";

--bun:split
DROP INDEX IF EXISTS "idx_agent_memories_tool";

--bun:split
DROP INDEX IF EXISTS "idx_agent_memories_active";

--bun:split
DROP TABLE IF EXISTS "agent_memories";
