CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_user_pinned" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "pinned", "last_message_at" DESC);
