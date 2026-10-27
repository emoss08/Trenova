SELECT awut.model_key, awut.provider_id, awut.tool_name, awut.cause, sum(awut.calls)
FROM ai_work_unit_tools awut
WHERE awut.organization_id = $1 AND awut.business_unit_id = $2 AND awut.finished_at >= $3 AND awut.task = 'AssistantChat'
GROUP BY awut.model_key, awut.provider_id, awut.tool_name, awut.cause
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1790000000
