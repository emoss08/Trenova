--keel:notx
-- Nova's bulk generation pages an organization's unreviewed invoice items oldest first, and the
-- schedule finds the organizations that have any: the index holds only that backlog, so it stays
-- the size of what Nova has left to read.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_billing_queue_items_nova_backlog" ON "billing_queue_items"("organization_id", "business_unit_id", "created_at", "id")
WHERE
    "agent_review_hash" IS NULL AND "bill_type" = 'Invoice' AND "lane" IN ('NeedsYou', 'Waiting', 'Ready');
