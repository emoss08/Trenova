--keel:notx
-- The journal entry picker matches the number with ILIKE, which the trigram index on the number
-- serves; the reversible picker's partial index takes the same expression, replacing the one on
-- LOWER("entry_number") that no read uses any more.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_reversible_entry_number" ON "journal_entries" USING gin("entry_number" gin_trgm_ops)
WHERE
    "status" = 'Posted' AND "is_posted" IS TRUE AND "is_reversal" IS FALSE AND "reversed_by_id" IS NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_reversible_number";
