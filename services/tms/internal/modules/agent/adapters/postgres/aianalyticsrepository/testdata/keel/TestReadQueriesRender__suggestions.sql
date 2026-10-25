SELECT as_.suggested_agent_id, coalesce(max(ad.name), $5), count(*) FILTER (WHERE as_.outcome = 'Switched'), count(*) FILTER (WHERE as_.outcome = 'SentThrough'), count(*) FILTER (WHERE as_.outcome = 'AutoSwitched'), count(*) FILTER (WHERE as_.outcome = 'Kept'), count(*) FILTER (WHERE as_.outcome IN ('PassedOver', 'Shown'))
FROM agent_suggestions as_
LEFT JOIN agent_definitions ad ON ad.business_unit_id = $1 AND ad.organization_id = $2 AND ad.id = as_.suggested_agent_id
WHERE as_.business_unit_id = $1 AND as_.organization_id = $2 AND as_.created_at >= $3 AND (as_.outcome <> 'Shown' OR as_.created_at < $4)
GROUP BY as_.suggested_agent_id
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = 0
-- $5 = ''
