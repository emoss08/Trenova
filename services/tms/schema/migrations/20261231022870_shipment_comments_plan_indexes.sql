--keel:notx
-- The customer tracking page reads a shipment's latest comments shared with the customer, and a
-- thread reads its replies oldest first: each page comes in order from its own index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_comments_customer_visible" ON "shipment_comments"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC)
WHERE
    "visibility" = 'Customer' AND "deleted_at" IS NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_comments_replies" ON "shipment_comments"("organization_id", "business_unit_id", "parent_comment_id", "created_at", "id")
WHERE
    "parent_comment_id" IS NOT NULL;
