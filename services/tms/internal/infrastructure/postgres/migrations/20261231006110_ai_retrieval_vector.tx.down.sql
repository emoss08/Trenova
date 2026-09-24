-- The extension itself is left installed: other objects may depend on it, and
-- removing it is an operator's decision rather than a schema rollback.
DROP TRIGGER IF EXISTS "trg_inbound_messages_ai_embeddings_purge" ON "inbound_messages";

--bun:split
DROP TRIGGER IF EXISTS "trg_documents_ai_embeddings_purge" ON "documents";

--bun:split
DROP TRIGGER IF EXISTS "trg_agent_memories_ai_embeddings_purge" ON "agent_memories";

--bun:split
DROP FUNCTION IF EXISTS ai_embeddings_purge_source();

--bun:split
DROP TABLE IF EXISTS "ai_catalog_embeddings";

--bun:split
DROP TABLE IF EXISTS "ai_embeddings";
