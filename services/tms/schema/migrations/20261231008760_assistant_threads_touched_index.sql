--keel:notx
-- Concurrent index builds cannot run inside the migration transaction.
-- The Desk rail reads each shelf of a person's conversations by (pinned, greatest(last_message_at,
-- created_at), id), newest first: the instant a conversation was last touched, which is also what
-- a shelf's range is drawn on. Scanned backwards this index seeks straight to a shelf's range and
-- to the cursor inside it, and the shelf counts read it without touching the table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_user_touched" ON "assistant_threads"(
    "organization_id",
    "business_unit_id",
    "user_id",
    "pinned",
    (greatest("last_message_at", "created_at")),
    "id"
);
