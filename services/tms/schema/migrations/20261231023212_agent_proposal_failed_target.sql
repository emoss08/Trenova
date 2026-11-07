--keel:notx
-- An approved coverage proposal whose assignment failed stays on its move in the
-- dispatch board, with the reason, until the move changes. The board reads each
-- move's latest failed proposal by it; this is that question's index: only failed
-- rows that name a record, tenant first, newest first.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_failed_target"
    ON "agent_proposals"("organization_id", "business_unit_id", "tool_name", "target_resource", "target_id", "created_at" DESC)
    WHERE "status" = 'ExecutionFailed' AND "target_id" IS NOT NULL;
