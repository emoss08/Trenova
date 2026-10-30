--keel:notx
-- The accounting mapping's active accounts probe each of the chart's accounts for its newest
-- posted line: the index walks one account's lines by journal entry, newest first, and returns the
-- entry each joins to without reading the line.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_entry_lines_account_entry" ON "journal_entry_lines"("organization_id", "business_unit_id", "gl_account_id", "journal_entry_id");

--keel:split
-- Every read of an account's lines is a tenant's, which the index above serves.
DROP INDEX CONCURRENTLY IF EXISTS "idx_journal_entry_lines_gl_account";
