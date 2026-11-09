--keel:notx
-- A factoring company's page reads its money path, its funding pace and its customers' funded
-- totals for one factor at a time: invoice states by factor and verdict (index-only, with the
-- balance and ready time), funded schedules by factor newest first, and funded schedule items by
-- customer.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_invoice_states_factor_verdict" ON "factoring_invoice_states"("organization_id", "business_unit_id", "factoring_company_id", "verdict") INCLUDE ("open_balance_minor", "ready_since")
WHERE "factoring_company_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_submissions_factor_funded" ON "factoring_submissions"("organization_id", "business_unit_id", "factoring_company_id", "funded_at" DESC, "id" DESC)
WHERE "funded_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_submission_items_funded_customer" ON "factoring_submission_items"("organization_id", "business_unit_id", "customer_id")
WHERE "status" = 'Funded';
