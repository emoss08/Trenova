--keel:notx
-- A fiscal period closes only when none of its batches' sources is unposted: the count reads the
-- period's batches from an index on the period, carrying the id the join to the sources needs,
-- rather than scanning every batch. The index also serves the period's foreign key.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_journal_batches_fiscal_period" ON "journal_batches"("organization_id", "business_unit_id", "fiscal_period_id", "id");

--keel:split
-- The billing queue pages its unposted items oldest first, and checks each item's shipment by key
-- when the list filters by the shipment's customer or delivery date: the partial index holds only
-- the unposted items, in page order, so the page stops at its limit.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_unposted_created" ON "billing_queue_items"("organization_id", "business_unit_id", "created_at", "id")
WHERE
    "status" <> 'Posted';

--keel:split
-- Settlement and pay event lists filtered by one status page newest first: the status leads the
-- order, so a rare status (Voided) walks its own rows in page order instead of being fetched and
-- sorted. The settlement index replaces the one on the status alone.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlements_status_created" ON "driver_settlements"("organization_id", "business_unit_id", "status", "created_at" DESC, "id" DESC);

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_driver_settlements_status";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_pay_events_status_created" ON "driver_pay_events"("organization_id", "business_unit_id", "status", "created_at" DESC, "id" DESC);

--keel:split
-- Factoring evaluation pages the invoices with a live readiness state by invoice id, merged with
-- the open factored invoices' page in id order: the partial index gives that order.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_factoring_invoice_states_candidates" ON "factoring_invoice_states"("organization_id", "business_unit_id", "invoice_id")
WHERE
    "verdict" <> 'Excluded';
