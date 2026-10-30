--keel:notx
-- The order list's capped total counts a tenant's orders, all or in the statuses its filter
-- lists: the index holds the tenant and the status, so both counts read it index-only, and its
-- repeated keys deduplicate it to a few pages.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_orders_tenant_status" ON "orders"("organization_id", "business_unit_id", "status");

--keel:split
-- The invoice list's capped total counts a tenant's invoices in the settlement statuses its filter
-- lists, index-only from the tenant and the settlement status.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invoices_tenant_settlement_status" ON "invoices"("organization_id", "business_unit_id", "settlement_status");
