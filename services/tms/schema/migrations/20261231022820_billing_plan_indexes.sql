--keel:notx
-- Factoring walks a tenant's open factored invoices by id, and the journal repair sweep walks
-- posted adjustment credit memos by id across tenants, checking each for a ledger line: both
-- sets are small beside the invoices, so partial indexes hold only them. Factoring's invoice
-- search matches the invoice number with ILIKE through a trigram index, and its queue pages
-- newest first without a verdict.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_factored_open" ON "invoices"("organization_id", "business_unit_id", "id")
WHERE
    "status" = 'Posted' AND "factoring_company_id" IS NOT NULL AND "bill_type" <> 'CreditMemo' AND "total_amount_minor" > "applied_amount_minor";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_adjustment_memos" ON "invoices"("id")
WHERE
    "is_adjustment_artifact" IS TRUE AND "bill_type" = 'CreditMemo' AND "status" = 'Posted' AND "source_invoice_adjustment_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_trgm_number" ON "invoices" USING gin("number" gin_trgm_ops);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_customer_ledger_entries_source_object" ON "customer_ledger_entries"("organization_id", "business_unit_id", "source_object_id");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_invoice_states_created" ON "factoring_invoice_states"("organization_id", "business_unit_id", "created_at" DESC, "invoice_id" DESC);
