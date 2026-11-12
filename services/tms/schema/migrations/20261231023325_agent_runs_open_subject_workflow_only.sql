--keel:notx
-- One open event or wait run per agent and subject holds for runs a workflow
-- carries. Inline runs (dispatch coverage, routines) are not a subject's run:
-- coverage refreshes replace each other's proposals, and routines dedupe on
-- the pending proposal itself.
DROP INDEX CONCURRENTLY IF EXISTS "uq_agent_runs_open_subject";

--keel:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_agent_runs_open_subject" ON "agent_runs"("organization_id", "business_unit_id", "agent_definition_id", "subject_id")
WHERE
    "trigger" IN ('Event', 'Wait')
    AND "agent_definition_id" IS NOT NULL
    AND "workflow_id" IS NOT NULL
    AND "status" IN ('Pending', 'GatheringContext', 'Diagnosing', 'AwaitingDecision');
