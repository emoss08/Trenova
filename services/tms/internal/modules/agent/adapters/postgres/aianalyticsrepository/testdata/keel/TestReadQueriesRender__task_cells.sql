SELECT awu.model_key, coalesce(awu.provider_id, $4), awu.task, count(*), count(*) FILTER (WHERE awu.outcome = 'Completed')
FROM ai_work_units awu
WHERE awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.task = 'AssistantChat' AND awu.finished_at >= $3
GROUP BY awu.model_key, coalesce(awu.provider_id, $4), awu.task
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
