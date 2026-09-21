DROP INDEX IF EXISTS "idx_assistant_threads_origin_last_message";

--bun:split
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "mentions";

--bun:split
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "attachments";
