-- The Desk's rail marks a conversation whose newest message arrived after its
-- owner last looked at it. A thread belongs to one person, so the marker lives
-- on the thread rather than in a per-reader table.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "last_read_at" bigint NOT NULL DEFAULT 0;

--bun:split
UPDATE "assistant_threads"
SET "last_read_at" = "last_message_at"
WHERE "last_read_at" = 0;

--bun:split
COMMENT ON COLUMN "assistant_threads"."last_read_at" IS 'When the owner last read the conversation; a message after it is unread';
