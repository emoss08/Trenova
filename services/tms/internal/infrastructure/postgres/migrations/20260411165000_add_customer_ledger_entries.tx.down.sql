DROP INDEX IF EXISTS idx_customer_ledger_entries_customer_date;

--bun:split
DROP TABLE IF EXISTS customer_ledger_entries;
