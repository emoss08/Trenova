-- Rolling back forgets each reply's provider id; the next call after it asks the
-- Claude API nothing to compare against.
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "provider_message_id";
