--keel:notx
-- The billing queue review reads a payer's latest posted invoices, newest first, with their
-- charges: the index walks one payer's invoices in date order and stops at the limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_posted_by_customer_date" ON "invoices"("organization_id", "business_unit_id", "customer_id", "invoice_date" DESC, "id" DESC)
WHERE
    "status" = 'Posted' AND "bill_type" = 'Invoice' AND "is_adjustment_artifact" IS FALSE;
