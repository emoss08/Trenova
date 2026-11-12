--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_recheck_due" ON "billing_queue_items"("organization_id", "business_unit_id", "recheck_due_at", "id")
WHERE
    "recheck_due_at" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_recheck_due_tenants" ON "billing_queue_items"("recheck_due_at")
WHERE
    "recheck_due_at" IS NOT NULL;
