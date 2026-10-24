-- The reversal picker lists a tenant's reversible entries newest first and searches their numbers.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_reversible" ON "journal_entries"("organization_id", "business_unit_id", "accounting_date" DESC, "id" DESC)
WHERE
    "status" = 'Posted' AND "is_posted" = true AND "is_reversal" = false AND "reversed_by_id" IS NULL;

--bun:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_reversible_number_trgm" ON "journal_entries" USING gin(LOWER("entry_number") gin_trgm_ops)
WHERE
    "status" = 'Posted' AND "is_posted" = true AND "is_reversal" = false AND "reversed_by_id" IS NULL;
