--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_lane" ON "billing_queue_items"("organization_id", "business_unit_id", "lane", "created_at", "id")
WHERE
    "lane" <> 'Closed';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_recheck" ON "billing_queue_items"("checked_at" NULLS FIRST, "id")
WHERE
    "lane" IN ('Checking', 'NeedsYou', 'Waiting', 'Ready');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_review_completed" ON "billing_queue_items"("organization_id", "business_unit_id", "review_completed_at")
WHERE
    "review_completed_at" IS NOT NULL;
