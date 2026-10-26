INSERT INTO ai_work_unit_tools (unit_id, tool_name, cause, finished_at, model_key, provider_id, agent_definition_id, task, calls, organization_id, business_unit_id)
SELECT s.*, $1, $2
FROM (
  SELECT ars.owner_id, ars.tool_name, c.cause, awu.finished_at, awu.model_key, coalesce(awu.provider_id, $7), coalesce(awu.agent_definition_id, $8), awu.task, count(*)
  FROM ai_work_units awu
  JOIN agent_run_steps ars ON ars.business_unit_id = $2 AND ars.organization_id = $1 AND ars.owner_id = awu.id
  CROSS JOIN LATERAL (
    SELECT CASE WHEN ars.status <> $9 THEN $10::text WHEN coalesce((ars.outcome #>> $11), $12) = ANY($13) THEN (ars.outcome #>> $11) WHEN ((ars.outcome #>> $14) = $15 AND (ars.outcome #>> $16) = $17) THEN $18::text WHEN (ars.outcome #>> $14) = $19 THEN $20::text WHEN (ars.outcome #>> $14) = $21 THEN $22::text WHEN ((ars.outcome #>> $14) = $23 AND (ars.outcome #>> $16) = $24) THEN $18::text WHEN (ars.outcome #>> $14) = ANY($25) THEN $26::text WHEN (lower(coalesce((ars.outcome #>> $27), $28)) LIKE $29 OR lower(coalesce((ars.outcome #>> $27), $28)) LIKE $30 OR lower(coalesce((ars.outcome #>> $27), $28)) LIKE $31) THEN $32::text ELSE $33::text END AS cause
  ) c
  WHERE awu.business_unit_id = $2 AND awu.organization_id = $1 AND awu.refresh_stamp = $3 AND awu.source IN ('AgentRun', 'AssistantTurn') AND ars.kind = $4 AND ars.status <> $5 AND ars.tool_name <> $6
  GROUP BY ars.owner_id, ars.tool_name, c.cause, awu.finished_at, awu.model_key, awu.provider_id, awu.agent_definition_id, awu.task
) s
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 7
-- $4 = 'Tool'
-- $5 = 'Started'
-- $6 = ''
-- $7 = ''
-- $8 = ''
-- $9 = 'Failed'
-- $10 = ''
-- $11 = '{cause}'
-- $12 = ''
-- $13 = '{WrongArguments,ToolTimedOut,RepeatedCall,BlockedByRule,ToolError,ToolNotHeld}'
-- $14 = '{verdict}'
-- $15 = 'invalid'
-- $16 = '{reason}'
-- $17 = 'no such tool is registered'
-- $18 = 'ToolNotHeld'
-- $19 = 'invalid'
-- $20 = 'WrongArguments'
-- $21 = 'duplicate'
-- $22 = 'RepeatedCall'
-- $23 = 'denied'
-- $24 = 'the agent does not hold the tool'
-- $25 = '{denied,over_budget}'
-- $26 = 'BlockedByRule'
-- $27 = '{reason}'
-- $28 = ''
-- $29 = '%deadline exceeded%'
-- $30 = '%timed out%'
-- $31 = '%timeout%'
-- $32 = 'ToolTimedOut'
-- $33 = 'ToolError'
