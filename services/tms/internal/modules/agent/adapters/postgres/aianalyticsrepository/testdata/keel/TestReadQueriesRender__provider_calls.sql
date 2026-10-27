SELECT aphs.model_key, aphs.provider_id, aphs.error_class, sum(aphs.calls), sum(aphs.fell_to_next), max(aphs.hour)
FROM ai_provider_hourly_stats aphs
WHERE aphs.organization_id = $1 AND aphs.business_unit_id = $2 AND aphs.hour >= $3 AND aphs.task = 'AssistantChat'
GROUP BY aphs.model_key, aphs.provider_id, aphs.error_class
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1790002800
