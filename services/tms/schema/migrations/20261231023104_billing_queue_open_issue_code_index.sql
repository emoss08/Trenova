--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_issues_open_code" ON "billing_queue_issues"("organization_id", "business_unit_id", "code", "item_id")
WHERE
    "resolution_key" IS NULL;
