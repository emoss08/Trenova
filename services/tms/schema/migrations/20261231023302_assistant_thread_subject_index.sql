--keel:notx
-- A conversation asked on a record's case is found again by the person and
-- the record, so a second question about the same record lands in the first
-- one's conversation rather than opening another.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_user_subject" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "subject_id", "last_message_at" DESC)
WHERE
    "subject_id" IS NOT NULL AND "status" = 'Active';
