CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_billing_queue_items_number" ON "billing_queue_items"("number");

--bun:split
DROP INDEX CONCURRENTLY IF EXISTS "uq_billing_queue_items_number_tenant";
