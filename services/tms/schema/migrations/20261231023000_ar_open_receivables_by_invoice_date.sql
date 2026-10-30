--keel:notx
-- The AR overview sums the open invoices by due date and dispute status, and the as-of history
-- starts from the invoices open now and dated by the as-of date. Both read the open invoices
-- index-only from this partial index, keyed by invoice date and carrying the amounts, due date and
-- dispute status, never the invoice heap.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_open_receivable_invoice_date" ON "invoices"("organization_id", "business_unit_id", "invoice_date") INCLUDE ("id", "due_date", "total_amount_minor", "applied_amount_minor", "dispute_status")
WHERE
    "status" = 'Posted' AND "bill_type" IN ('Invoice', 'DebitMemo') AND "total_amount_minor" > "applied_amount_minor";
