--keel:notx
-- Concurrent index drops cannot run inside the migration transaction.
-- The rail no longer pages by (pinned, last_message_at, created_at, id);
-- idx_assistant_threads_user_touched replaces this index, so it only costs writes.
DROP INDEX CONCURRENTLY IF EXISTS "idx_assistant_threads_user_rail";
