--keel:notx
-- Orders, service failures and capture stacks list newest first, and the order count a list
-- asks for stops at 10,001 rows: each reads the tenant's rows in that order from an index
-- instead of sorting or scanning them. The order index carries the status a status filter
-- counts by.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_orders_created" ON "orders"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC) INCLUDE ("status");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_sf_tenant_created" ON "service_failures"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_capture_batches_created" ON "capture_batches"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);
