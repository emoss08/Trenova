-- Concurrent index builds cannot run inside the migration transaction.
-- The Desk rail pages a person's conversations by (pinned, last_message_at, created_at, id),
-- newest first, continuing below the last row it holds. Scanned backwards this index returns
-- that order and seeks straight to the cursor, so a page costs its own rows however deep it is.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_user_rail" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "pinned", "last_message_at", "created_at", "id");
