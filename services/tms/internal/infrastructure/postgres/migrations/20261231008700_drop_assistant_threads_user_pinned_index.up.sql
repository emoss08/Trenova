-- Concurrent index drops cannot run inside the migration transaction.
-- idx_assistant_threads_user_rail leads with the same columns, so this one only costs writes.
DROP INDEX CONCURRENTLY IF EXISTS "idx_assistant_threads_user_pinned";
