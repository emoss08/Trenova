SELECT aphs.model_key, aphs.provider_id, (b.idx - 1), sum(b.v)
FROM ai_provider_hourly_stats aphs
CROSS JOIN LATERAL (
  SELECT unnest(aphs.first_token_hist) AS v, generate_series(1, cardinality(aphs.first_token_hist)) AS idx
) b
WHERE aphs.organization_id = $1 AND aphs.business_unit_id = $2 AND aphs.hour >= $3 AND aphs.task = 'AssistantChat' AND aphs.error_class = '' AND b.v > $4
GROUP BY aphs.model_key, aphs.provider_id, 3
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1790002800
-- $4 = 0
