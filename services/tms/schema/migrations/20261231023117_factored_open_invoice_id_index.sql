--keel:notx
-- The factoring candidate sweep pages open factored invoices by id. The index it walked carried
-- the customer for a read that the per-customer index now serves, and that width let the planner
-- prefer reading the per-factor index and sorting it; the narrowest index in id order is the
-- cheapest read on any data set.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_factored_open" ON "invoices"("organization_id", "business_unit_id", "id")
WHERE
    "status" = 'Posted' AND "factoring_company_id" IS NOT NULL AND "bill_type" <> 'CreditMemo' AND "total_amount_minor" > "applied_amount_minor";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_invoices_factored_open_customer";
