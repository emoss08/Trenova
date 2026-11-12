--keel:notx
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_agent_runs_open_subject" ON "agent_runs"("organization_id", "business_unit_id", "agent_definition_id", "subject_id")
WHERE
    "trigger" IN ('Event', 'Wait')
    AND "agent_definition_id" IS NOT NULL
    AND "status" IN ('Pending', 'GatheringContext', 'Diagnosing', 'AwaitingDecision');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_awaiting_decision" ON "agent_runs"("updated_at", "id")
WHERE
    "status" = 'AwaitingDecision'
    AND "workflow_id" IS NOT NULL;
