-- A coverage proposal names the move it would assign. The dispatch board reads
-- each move's pending proposal by it, and a plan run again supersedes what an
-- earlier plan proposed for the same moves by it, so a move carries at most one
-- pending proposal of a tool. This is that question's index: only pending rows
-- that name a record, tenant first.
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_pending_target"
    ON "agent_proposals"("organization_id", "business_unit_id", "tool_name", "target_resource", "target_id")
    WHERE "status" = 'Pending' AND "target_id" IS NOT NULL;
