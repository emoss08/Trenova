SELECT aphs.provider_id, aphs.hour, sum(aphs.calls), coalesce(sum(aphs.calls) FILTER (WHERE aphs.error_class IN ('RateLimited', 'TimedOut', 'CouldNotConnect', 'AccessDenied', 'ServerError')), $5), coalesce(sum(aphs.calls) FILTER (WHERE aphs.error_class = 'AccessDenied'), $6)
FROM ai_provider_hourly_stats aphs
WHERE aphs.organization_id = $1 AND aphs.business_unit_id = $2 AND aphs.hour >= $3 AND aphs.task = 'AssistantChat' AND aphs.provider_id <> $4
GROUP BY aphs.provider_id, aphs.hour
ORDER BY aphs.provider_id ASC, aphs.hour ASC
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1790002800
-- $4 = ''
-- $5 = 0
-- $6 = 0
