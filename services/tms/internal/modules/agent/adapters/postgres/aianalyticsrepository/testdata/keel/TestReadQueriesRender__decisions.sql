SELECT awu.model_key, coalesce(awu.provider_id, $4), count(*) FILTER (WHERE ad.decision = 'Accepted'), count(*) FILTER (WHERE ad.decision = 'Modified'), count(*) FILTER (WHERE ad.decision = 'Rejected')
FROM agent_decisions ad
JOIN agent_proposals ap ON ap.business_unit_id = $1 AND ap.organization_id = $2 AND ap.id = ad.proposal_id
JOIN ai_work_units awu ON awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.decision_run_id = ap.run_id
WHERE ad.business_unit_id = $1 AND ad.organization_id = $2 AND awu.task = 'AssistantChat' AND ad.created_at >= $3 AND ad.proposal_id IS NOT NULL AND ad.undone_at IS NULL
GROUP BY awu.model_key, 2
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
