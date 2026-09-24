DROP TRIGGER IF EXISTS "trg_inbound_messages_ai_index_entries_purge" ON "inbound_messages";

--bun:split
DROP TRIGGER IF EXISTS "trg_documents_ai_index_entries_purge" ON "documents";

--bun:split
DROP TRIGGER IF EXISTS "trg_agent_memories_ai_index_entries_purge" ON "agent_memories";

--bun:split
DROP FUNCTION IF EXISTS ai_index_entries_purge_source();

--bun:split
DROP TABLE IF EXISTS "ai_retrieval_settings";

--bun:split
DROP TABLE IF EXISTS "ai_index_entries";
