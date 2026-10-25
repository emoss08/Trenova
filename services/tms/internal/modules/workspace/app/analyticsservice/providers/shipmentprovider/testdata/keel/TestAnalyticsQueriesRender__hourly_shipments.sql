SELECT CAST(date_part('hour', timezone($5, to_timestamp(s.created_at))) AS bigint), CAST(count(*) AS double precision)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND (s.created_at >= $3 AND s.created_at <= $4)
GROUP BY 1
ORDER BY 1 ASC
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 'America/Chicago'
