DROP INDEX IF EXISTS "idx_assistant_messages_refused";

DROP INDEX IF EXISTS "idx_assistant_messages_thread";

DROP INDEX IF EXISTS "uq_assistant_messages_sequence";

--bun:split
DROP TABLE IF EXISTS "assistant_messages";

--bun:split
DROP INDEX IF EXISTS "idx_assistant_threads_agent";

DROP INDEX IF EXISTS "idx_assistant_threads_user";

--bun:split
DROP TABLE IF EXISTS "assistant_threads";
