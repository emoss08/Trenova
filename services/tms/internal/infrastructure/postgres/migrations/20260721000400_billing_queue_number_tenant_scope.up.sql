-- Invoice numbers are generated per organization, so the uniqueness guarantee must
-- be tenant-scoped: two organizations can legitimately mint the same formatted
-- number.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_billing_queue_items_number_tenant" ON "billing_queue_items"("number", "organization_id", "business_unit_id");

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "uq_billing_queue_items_number";
