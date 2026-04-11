DROP INDEX IF EXISTS idx_gl_account_balances_period;

--bun:split
DROP INDEX IF EXISTS idx_journal_sources_object;

--bun:split
DROP TABLE IF EXISTS "gl_account_balances_by_period";

--bun:split
DROP TABLE IF EXISTS "source_journal_links";

--bun:split
DROP TABLE IF EXISTS "journal_sources";
