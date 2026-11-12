-- A subject keeps one open event or wait run per agent. That used to be held
-- by the run's workflow id while the workflow stayed open; the workflow now
-- ends once its proposals are filed, so the index holds it. Older duplicates
-- left open by a lost workflow are closed first so the index can be built.
WITH ranked AS (
    SELECT
        "id",
        row_number() OVER (PARTITION BY "organization_id", "business_unit_id", "agent_definition_id", "subject_id" ORDER BY "created_at" DESC, "id" DESC) AS "rank"
    FROM
        "agent_runs"
    WHERE
        "trigger" IN ('Event', 'Wait')
        AND "agent_definition_id" IS NOT NULL
        AND "status" IN ('Pending', 'GatheringContext', 'Diagnosing', 'AwaitingDecision'))
UPDATE
    "agent_runs" AS r
SET
    "status" = 'Failed',
    "error_message" = 'A newer run of this agent opened for the same subject.',
    "completed_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
FROM
    ranked
WHERE
    ranked."id" = r."id"
    AND ranked."rank" > 1;
