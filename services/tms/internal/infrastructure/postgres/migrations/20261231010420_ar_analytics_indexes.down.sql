DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_ledger_entries_source_created";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_ledger_entries_related_invoice";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_credit_memo_applications_unapplied";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_credit_memo_applications_created";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_payments_posted_date";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_payments_reversed";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_payment_applications_created";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_invoices_receivable_invoice_date";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_invoices_receivable_due";

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_invoices_open_receivable";
