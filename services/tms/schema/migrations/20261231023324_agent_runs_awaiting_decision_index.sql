--keel:notx
-- The expiry sweep closes waiting runs that close on decision: one a workflow
-- carried, or one an event started (a routine's run carries no workflow).
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_awaiting_decision" ON "agent_runs"("updated_at", "id")
WHERE
    "status" = 'AwaitingDecision'
    AND ("workflow_id" IS NOT NULL OR "trigger" = 'Event');
