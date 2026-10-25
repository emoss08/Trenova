SELECT awu.model_key, coalesce(awu.provider_id, $4), count(*), count(*) FILTER (WHERE af.rating > $5)
FROM ai_feedback af
JOIN ai_work_units awu ON awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.id = coalesce(af.turn_id, af.run_id)
WHERE af.business_unit_id = $1 AND af.organization_id = $2 AND awu.task = 'AssistantChat' AND af.created_at >= $3
GROUP BY awu.model_key, coalesce(awu.provider_id, $4)
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
-- $5 = 0
