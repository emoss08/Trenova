--keel:notx
-- An agent that runs on its own reaches the same write again on its next run
-- while the first is still waiting on a person. The recorder reads what the
-- agent has waiting from the tools it just called, so the repeat adds nothing
-- to the queue. This is that read's index: pending rows by tool, tenant first.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_pending_tool" ON "agent_proposals"("organization_id", "business_unit_id", "tool_name", "created_at" DESC)
WHERE
    "status" = 'Pending';
