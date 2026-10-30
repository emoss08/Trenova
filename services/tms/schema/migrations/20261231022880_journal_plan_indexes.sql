--keel:notx
-- A journal entry's sources are read, counted and restamped by the entry, newest or oldest first,
-- and an object's sources are read in the order they were recorded: both orders come from an
-- index, the object one replacing the index that found the rows but not their order. A period
-- closes only when none of its sources is still unposted, a small set a partial index holds. An
-- entry's links are counted by the entry.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_sources_entry_created" ON "journal_sources"("organization_id", "business_unit_id", "journal_entry_id", "created_at", "id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_sources_object_created" ON "journal_sources"("organization_id", "business_unit_id", "source_object_type", "source_object_id", "created_at");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_sources_object";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_sources_unposted_batch" ON "journal_sources"("organization_id", "business_unit_id", "journal_batch_id")
WHERE
    "status" <> 'Posted';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_source_journal_links_entry" ON "source_journal_links"("organization_id", "business_unit_id", "journal_entry_id");

--keel:split
-- Journal entry reads write their flags as IS TRUE and IS FALSE, which Postgres does not prove
-- from a predicate written as = true, so the partial indexes those reads need say IS TRUE and
-- IS FALSE too: posted entries by accounting date (the ledger) and by posting time (the
-- accounting sync candidates), entries awaiting review by accounting date, and the reversible
-- entries a picker pages and searches by number. The indexes written with = true are replaced.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_posted_date" ON "journal_entries"("organization_id", "business_unit_id", "accounting_date")
WHERE
    "is_posted" IS TRUE;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_posted_accounting_date";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_posted_order" ON "journal_entries"("organization_id", "business_unit_id", "posted_at", "id")
WHERE
    "is_posted" IS TRUE;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_review" ON "journal_entries"("organization_id", "business_unit_id", "accounting_date", "id")
WHERE
    "status" IN ('Pending', 'Approved') AND "is_posted" IS FALSE;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_reversible_date" ON "journal_entries"("organization_id", "business_unit_id", "accounting_date" DESC, "id" DESC)
WHERE
    "status" = 'Posted' AND "is_posted" IS TRUE AND "is_reversal" IS FALSE AND "reversed_by_id" IS NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_reversible";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_reversible_number" ON "journal_entries" USING gin(LOWER("entry_number") gin_trgm_ops)
WHERE
    "status" = 'Posted' AND "is_posted" IS TRUE AND "is_reversal" IS FALSE AND "reversed_by_id" IS NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_reversible_number_trgm";

--keel:split
-- The journal entry list searches the number, description, reference id and reference type with
-- ILIKE: each column has a trigram index, so the search is a bitmap of the four.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_trgm_entry_number" ON "journal_entries" USING gin("entry_number" gin_trgm_ops);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_trgm_description" ON "journal_entries" USING gin("description" gin_trgm_ops);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_trgm_reference_id" ON "journal_entries" USING gin("reference_id" gin_trgm_ops);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entries_trgm_reference_type" ON "journal_entries" USING gin("reference_type" gin_trgm_ops);

--keel:split
-- Reversals page newest first, filtered by status.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_reversals_created" ON "journal_reversals"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC) INCLUDE ("status");
