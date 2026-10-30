--keel:notx
-- A factoring evaluation targeted at one customer or one factor pages that payer's open factored
-- invoices by id: each partial index holds only open factored invoices, keyed so the page walks
-- one payer's invoices in id order and stops at its limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_factored_open_by_customer" ON "invoices"("organization_id", "business_unit_id", "customer_id", "id")
WHERE
    "status" = 'Posted' AND "factoring_company_id" IS NOT NULL AND "bill_type" <> 'CreditMemo' AND "total_amount_minor" > "applied_amount_minor";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_factored_open_by_factor" ON "invoices"("organization_id", "business_unit_id", "factoring_company_id", "id")
WHERE
    "status" = 'Posted' AND "factoring_company_id" IS NOT NULL AND "bill_type" <> 'CreditMemo' AND "total_amount_minor" > "applied_amount_minor";
