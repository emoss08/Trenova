SELECT awut.cause, awut.provider_id, sum(awut.calls)
FROM ai_work_unit_tools awut
WHERE awut.organization_id = $1 AND awut.business_unit_id = $2 AND awut.finished_at >= $3 AND awut.task = 'AssistantChat' AND awut.tool_name = $4 AND awut.provider_id = $5
GROUP BY awut.cause, awut.provider_id
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1790000000
-- $4 = 'search'
-- $5 = 'aip_1'
