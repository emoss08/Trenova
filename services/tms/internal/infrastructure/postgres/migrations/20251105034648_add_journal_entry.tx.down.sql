SET statement_timeout = 0;

DROP TRIGGER IF EXISTS journal_entries_search_update ON "journal_entries";

--bun:split
DROP FUNCTION IF EXISTS journal_entries_search_trigger() CASCADE;

--bun:split
DROP TABLE IF EXISTS "journal_entry_lines" CASCADE;

--bun:split
DROP TABLE IF EXISTS "journal_entries" CASCADE;

--bun:split
DROP TYPE IF EXISTS journal_entry_status_enum CASCADE;

--bun:split
DROP TYPE IF EXISTS journal_entry_type_enum CASCADE;
