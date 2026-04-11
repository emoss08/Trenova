ALTER TABLE "manual_journal_requests"
    DROP CONSTRAINT IF EXISTS "fk_manual_journal_requests_posted_batch";

--bun:split
DROP INDEX IF EXISTS idx_journal_entries_batch_id;

--bun:split
ALTER TABLE "journal_entries"
    DROP CONSTRAINT IF EXISTS "fk_journal_entries_batch";

--bun:split
ALTER TABLE "journal_entries"
    DROP COLUMN IF EXISTS "batch_id";

--bun:split
DROP TABLE IF EXISTS "journal_batches";
