SELECT awut.organization_id, awut.business_unit_id, awut.unit_id, awut.tool_name, awut.cause, awut.finished_at, awut.model_key, awut.provider_id, awut.agent_definition_id, awut.task, awut.calls
FROM ai_work_unit_tools awut
JOIN ai_work_units awu ON awu.business_unit_id = $2 AND awu.organization_id = $1 AND awu.id = awut.unit_id
WHERE awut.organization_id = $1 AND awut.business_unit_id = $2 AND awu.refresh_stamp = $3 AND awu.source IN ('AgentRun', 'AssistantTurn')
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 7
