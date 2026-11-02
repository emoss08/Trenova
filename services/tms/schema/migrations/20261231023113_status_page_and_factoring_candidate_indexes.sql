--keel:notx
-- Settlement and pay event lists filtered to one status page newest first: the status sits
-- between the tenant and the page order, so the page reads off the index in order. The
-- settlement index replaces idx_driver_settlements_status, its prefix.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlements_status_created" ON "driver_settlements"("organization_id", "business_unit_id", "status", "created_at" DESC, "id" DESC);

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_driver_settlements_status";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_pay_events_status_created" ON "driver_pay_events"("organization_id", "business_unit_id", "status", "created_at" DESC, "id" DESC);

--keel:split
-- The factoring candidate sweep's second branch pages the invoice states that are not excluded
-- by invoice id, as the open factored invoice branch pages idx_invoices_factored_open_customer.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_invoice_states_candidates" ON "factoring_invoice_states"("organization_id", "business_unit_id", "invoice_id")
WHERE
    "verdict" <> 'Excluded';
