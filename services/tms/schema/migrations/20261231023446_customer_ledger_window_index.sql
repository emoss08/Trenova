--keel:notx
-- A customer's ledger is read a window at a time: entries between two dates in (date, line, id)
-- order after a keyset position, and the amounts of the opening month's entries before a
-- statement starts. The index carries the whole order and the amount, so a page is one range scan
-- and the opening month's sum reads the index alone. It replaces the (customer, date) index, which
-- is its prefix.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_customer_window" ON "customer_ledger_entries"("organization_id", "business_unit_id", "customer_id", "transaction_date", "line_number", "id") INCLUDE ("amount_minor");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_ledger_entries_customer_date";
