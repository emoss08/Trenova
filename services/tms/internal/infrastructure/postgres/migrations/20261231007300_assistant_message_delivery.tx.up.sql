-- How a reply was delivered, so a conversation read back later says what the
-- reader saw at the time: a reply that stopped partway, and a reply another
-- model gave because the one asked first did not answer.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "truncated" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "fallback_from" JSONB;

--bun:split
COMMENT ON COLUMN "assistant_messages"."truncated" IS 'The provider stopped partway through this reply; the content is what arrived';

--bun:split
COMMENT ON COLUMN "assistant_messages"."fallback_from" IS 'The provider asked first, and why it did not answer, when another provider gave this reply';
