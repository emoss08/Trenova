DROP INDEX IF EXISTS "idx_inbound_messages_search_vector";

--bun:split
ALTER TABLE "inbound_messages"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
DROP FUNCTION IF EXISTS mail_own_words(text);
