SELECT DISTINCT aphs.model_key, aphs.provider_id, aphs.organization_id, aphs.business_unit_id, aphs.hour, aphs.model, aphs.task, aphs.agent_definition_id, aphs.error_class
FROM ai_provider_hourly_stats aphs
WHERE aphs.organization_id = $1 AND aphs.business_unit_id = $2 AND aphs.hour >= $3
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1789999200
