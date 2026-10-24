--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_open_receivable"
    ON "invoices"("organization_id", "business_unit_id", "due_date")
    WHERE "status" = 'Posted'
      AND "bill_type" IN ('Invoice', 'DebitMemo')
      AND "total_amount_minor" > "applied_amount_minor";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_receivable_due"
    ON "invoices"("organization_id", "business_unit_id", "due_date")
    WHERE "status" = 'Posted'
      AND "bill_type" IN ('Invoice', 'DebitMemo');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_receivable_invoice_date"
    ON "invoices"("organization_id", "business_unit_id", "invoice_date")
    WHERE "bill_type" IN ('Invoice', 'DebitMemo');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_payment_applications_created"
    ON "customer_payment_applications"("organization_id", "business_unit_id", "created_at")
    INCLUDE ("invoice_id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_payments_reversed"
    ON "customer_payments"("organization_id", "business_unit_id", "reversed_at")
    WHERE "reversed_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_payments_posted_date"
    ON "customer_payments"("organization_id", "business_unit_id", "payment_date")
    WHERE "status" = 'Posted';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_credit_memo_applications_created"
    ON "credit_memo_applications"("organization_id", "business_unit_id", "created_at")
    INCLUDE ("invoice_id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_credit_memo_applications_unapplied"
    ON "credit_memo_applications"("organization_id", "business_unit_id", "unapplied_at")
    INCLUDE ("invoice_id")
    WHERE "unapplied_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_related_invoice"
    ON "customer_ledger_entries"("organization_id", "business_unit_id", "related_invoice_id", "source_object_type")
    WHERE "related_invoice_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_source_created"
    ON "customer_ledger_entries"("organization_id", "business_unit_id", "source_object_type", "created_at")
    INCLUDE ("related_invoice_id")
    WHERE "related_invoice_id" IS NOT NULL;
