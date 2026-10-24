DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_reversible_number_trgm";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entries_reversible";
