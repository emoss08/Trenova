INSERT INTO ai_provider_hourly_stats (hour, provider_id, model, model_key, task, agent_definition_id, error_class, calls, fell_to_next, first_token_hist, organization_id, business_unit_id)
SELECT s.*, $1, $2
FROM (
  WITH calls (hour, provider_id, model, task, agent_definition_id, error_class, succeeded, first_token_ms, trace_id) AS (
    SELECT ((aur.created_at / $7::bigint) * $7::bigint), coalesce(aur.provider_id, $8), aur.model, aur.task, coalesce(aur.agent_definition_id, $9), CASE WHEN aur.succeeded IS TRUE THEN $3::text WHEN aur.error_class = $10 THEN $11::text WHEN aur.error_class = $12 THEN $13::text WHEN (aur.error_class = $14 OR aur.error_message ~* $15) THEN $16::text WHEN (aur.error_class = $17 OR aur.error_message ~* $18) THEN $19::text WHEN aur.error_class = $20 THEN $21::text WHEN aur.error_class = $22 THEN $23::text ELSE $24::text END, aur.succeeded, aur.first_token_ms, aur.trace_id
    FROM ai_usage_records aur
    WHERE aur.business_unit_id = $2 AND aur.organization_id = $1 AND aur.created_at >= $5 AND aur.created_at < $6 AND aur.surface NOT IN ('Indexing', 'Retrieval')
  ), rescued (trace_id, providers) AS (
    SELECT c.trace_id, array_agg(DISTINCT c.provider_id)
    FROM calls c
    WHERE c.succeeded IS TRUE AND c.trace_id IS NOT NULL
    GROUP BY c.trace_id
  ), grouped (hour, provider_id, model, task, agent_definition_id, error_class, calls, fell_to_next) AS (
    SELECT c2.hour, c2.provider_id, c2.model, c2.task, c2.agent_definition_id, c2.error_class, count(*), count(*) FILTER (WHERE (c2.succeeded = $25 AND r.providers IS NOT NULL AND NOT ((r.providers <@ ARRAY[c2.provider_id]))))
    FROM calls c2
    LEFT JOIN rescued r ON r.trace_id = c2.trace_id
    GROUP BY c2.hour, c2.provider_id, c2.model, c2.task, c2.agent_definition_id, c2.error_class
  ), hist (hour, provider_id, model, task, agent_definition_id, bucket, n) AS (
    SELECT c3.hour, c3.provider_id, c3.model, c3.task, c3.agent_definition_id, CASE WHEN c3.first_token_ms < 16 THEN 0 ELSE least(63, (CAST(floor((('4'::double precision * ln((CAST(c3.first_token_ms AS double precision) / '16'::double precision))) / ln('2'::double precision))) AS bigint) + 1)) END, count(*)
    FROM calls c3
    WHERE c3.succeeded IS TRUE AND c3.first_token_ms IS NOT NULL
    GROUP BY c3.hour, c3.provider_id, c3.model, c3.task, c3.agent_definition_id, CASE WHEN c3.first_token_ms < 16 THEN 0 ELSE least(63, (CAST(floor((('4'::double precision * ln((CAST(c3.first_token_ms AS double precision) / '16'::double precision))) / ln('2'::double precision))) AS bigint) + 1)) END
  ), hist_keys (hour, provider_id, model, task, agent_definition_id) AS (
    SELECT DISTINCT h.hour, h.provider_id, h.model, h.task, h.agent_definition_id
    FROM hist h
  ), arrays (hour, provider_id, model, task, agent_definition_id, first_token_hist) AS (
    SELECT hk.hour, hk.provider_id, hk.model, hk.task, hk.agent_definition_id, array_agg(coalesce(h2.n, $26) ORDER BY b.i ASC)
    FROM hist_keys hk
    CROSS JOIN LATERAL (SELECT generate_series(0, 63) AS i) b
    LEFT JOIN hist h2 ON h2.hour = hk.hour AND h2.provider_id = hk.provider_id AND h2.model = hk.model AND h2.task = hk.task AND h2.agent_definition_id = hk.agent_definition_id AND h2.bucket = b.i
    GROUP BY hk.hour, hk.provider_id, hk.model, hk.task, hk.agent_definition_id
  )
  SELECT g.hour, g.provider_id, g.model, $3::text, g.task, g.agent_definition_id, g.error_class, g.calls, g.fell_to_next, CASE WHEN g.error_class = $4 THEN coalesce(a.first_token_hist, '{}') ELSE '{}' END
  FROM grouped g
  LEFT JOIN arrays a ON a.hour = g.hour AND a.provider_id = g.provider_id AND a.model = g.model AND a.task = g.task AND a.agent_definition_id = g.agent_definition_id
) s
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = ''
-- $4 = ''
-- $5 = 1789996400
-- $6 = 1790003600
-- $7 = 3600
-- $8 = ''
-- $9 = ''
-- $10 = 'cancelled'
-- $11 = 'Cancelled'
-- $12 = 'refused'
-- $13 = 'Declined'
-- $14 = 'auth'
-- $15 = '(\m401\M|\m403\M|unauthori[sz]ed|forbidden|access denied|invalid api key|invalid x-api-key|authentication|permission denied|not authorized)'
-- $16 = 'AccessDenied'
-- $17 = 'rate_limited'
-- $18 = '(\m429\M|rate.?limit|too many requests|quota exceeded|throttl)'
-- $19 = 'RateLimited'
-- $20 = 'timeout'
-- $21 = 'TimedOut'
-- $22 = 'network'
-- $23 = 'CouldNotConnect'
-- $24 = 'ServerError'
-- $25 = false
-- $26 = 0
