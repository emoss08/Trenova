SELECT s.customer_id, coalesce(c.name, $5), CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $6), $7) AS double precision), count(*) FILTER (WHERE (s.created_at >= $6)), CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE (s.created_at >= $3 AND s.created_at < $8)), $9) AS double precision), CAST(sum(CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $6), $7) AS double precision)) OVER () AS double precision)
FROM shipments s
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s.customer_id
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND (s.created_at >= $3 AND s.created_at <= $4) AND s.status <> 'Canceled'
GROUP BY s.customer_id, c.name
HAVING (CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $6), $7) AS double precision) > $10)
ORDER BY 3 DESC LIMIT 5
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 500
-- $4 = 1500
-- $5 = ''
-- $6 = 1000
-- $7 = 0
-- $8 = 1000
-- $9 = 0
-- $10 = 0
