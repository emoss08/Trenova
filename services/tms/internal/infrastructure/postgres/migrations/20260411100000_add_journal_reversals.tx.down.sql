DROP INDEX IF EXISTS uq_journal_reversals_original_active;

--bun:split
DROP TABLE IF EXISTS journal_reversals;

--bun:split
DROP TYPE IF EXISTS journal_reversal_status_enum;
