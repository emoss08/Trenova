--keel:notx
-- The AR history as of a date walks back from each invoice's stored applied amount through the
-- settlements made since the window start, so it reads payment applications, credit memo
-- applications and factoring ledger rows by tenant and creation time alone, never an invoice's
-- whole history. Each index carries every column the walk-back takes, so the read is index-only,
-- and replaces the index without them. Factoring ledger rows are no longer read by invoice.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_payment_applications_created_settled" ON "customer_payment_applications"("organization_id", "business_unit_id", "created_at") INCLUDE ("invoice_id", "customer_payment_id", "applied_amount_minor", "short_pay_amount_minor");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_payment_applications_created";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_credit_memo_applications_created_settled" ON "credit_memo_applications"("organization_id", "business_unit_id", "created_at") INCLUDE ("invoice_id", "unapplied_at", "applied_amount_minor");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_credit_memo_applications_created";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_source_created_amount" ON "customer_ledger_entries"("organization_id", "business_unit_id", "source_object_type", "created_at") INCLUDE ("related_invoice_id", "amount_minor")
WHERE
    "related_invoice_id" IS NOT NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_ledger_entries_source_created";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_customer_ledger_entries_factoring_invoice";
