--keel:notx
-- The accounting sync pages each kind of sales document posted or voided by posting time, and the
-- invoice list pages newest first, filtered by settlement status: both orders come from an index.
-- Open receivables are read by customer and as a whole (aging, the overview, the worklist, the
-- collections row, the as-of history), so a partial index over the open invoices carries every
-- column those reads take and serves them as index-only scans. Posted receivables are summed by
-- invoice date and checked for existence; invoices are found by customer; factoring's open
-- factored invoices carry their customer, replacing the index without it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_sync_posted" ON "invoices"("organization_id", "business_unit_id", "bill_type", "posted_at", "id")
WHERE
    "status" IN ('Posted', 'Voided') AND "posted_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_created" ON "invoices"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC) INCLUDE ("settlement_status");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_open_receivable_customer" ON "invoices"("organization_id", "business_unit_id", "customer_id") INCLUDE ("id", "bill_to_name", "number", "invoice_date", "due_date", "total_amount_minor", "applied_amount_minor", "dispute_status", "factoring_company_id", "factoring_agreement_id")
WHERE
    "status" = 'Posted' AND "bill_type" IN ('Invoice', 'DebitMemo') AND "total_amount_minor" > "applied_amount_minor";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_posted_receivable" ON "invoices"("organization_id", "business_unit_id", "invoice_date") INCLUDE ("customer_id", "total_amount_minor", "voided_at")
WHERE
    "bill_type" IN ('Invoice', 'DebitMemo') AND "posted_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_customer" ON "invoices"("organization_id", "business_unit_id", "customer_id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_factored_open_customer" ON "invoices"("organization_id", "business_unit_id", "id") INCLUDE ("customer_id")
WHERE
    "status" = 'Posted' AND "factoring_company_id" IS NOT NULL AND "bill_type" <> 'CreditMemo' AND "total_amount_minor" > "applied_amount_minor";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_invoices_factored_open";

--keel:split
-- Short pays are rare: the open item and worklist reads check an invoice for one through a partial
-- index of them. The ledger balance as of a date is an index-only sum by transaction date, and the
-- as-of history reads factoring ledger rows by invoice from a partial index of them.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_payment_applications_short_pay" ON "customer_payment_applications"("organization_id", "business_unit_id", "invoice_id") INCLUDE ("customer_payment_id")
WHERE
    "short_pay_amount_minor" > 0;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_transaction_date" ON "customer_ledger_entries"("organization_id", "business_unit_id", "transaction_date") INCLUDE ("amount_minor");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_factoring_invoice" ON "customer_ledger_entries"("organization_id", "business_unit_id", "related_invoice_id") INCLUDE ("created_at", "amount_minor")
WHERE
    "source_object_type" = 'FactoringSubmissionItem' AND "related_invoice_id" IS NOT NULL;

--keel:split
-- The billing queue counts the items after a cursor that are not posted: the created index carries
-- the status, so the count reads the index alone. It replaces the index without it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_created_status" ON "billing_queue_items"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC) INCLUDE ("status");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_billing_queue_items_tenant_created";
