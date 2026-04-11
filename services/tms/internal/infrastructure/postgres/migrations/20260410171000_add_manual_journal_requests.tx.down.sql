DROP INDEX IF EXISTS idx_manual_journal_request_lines_request;

--bun:split
DROP INDEX IF EXISTS idx_manual_journal_requests_fiscal_period;

--bun:split
DROP INDEX IF EXISTS idx_manual_journal_requests_status;

--bun:split
DROP TABLE IF EXISTS manual_journal_request_lines;

--bun:split
DROP TABLE IF EXISTS manual_journal_requests;

--bun:split
DROP TYPE IF EXISTS manual_journal_request_status_enum;
