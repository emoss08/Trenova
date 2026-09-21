-- A person can hand the assistant a file and name a record from the composer.
-- Both are kept on the user turn, beside the page context, so an answer can
-- be read against what it was given.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "attachments" jsonb;

--bun:split
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "mentions" jsonb;

--bun:split
-- A quick question from the palette lives on a hidden thread until it is
-- kept; the sweep that removes stale ones reads by origin and recency.
CREATE INDEX IF NOT EXISTS "idx_assistant_threads_origin_last_message" ON "assistant_threads"("origin", "last_message_at");
