WITH customer_mix (customer_id, name, revenue, loads, previous_revenue, total_revenue, row_rank) AS (
  SELECT s.customer_id, coalesce(c.name, $6), CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $7), $8) AS double precision), count(*) FILTER (WHERE (s.created_at >= $7)), CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE (s.created_at >= $4 AND s.created_at < $9)), $10) AS double precision), CAST(sum(CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $7), $8) AS double precision)) OVER () AS double precision), row_number() OVER (ORDER BY CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $7), $8) AS double precision) DESC)
  FROM shipments s
  JOIN customers c ON c.business_unit_id = $2 AND c.organization_id = $3 AND c.id = s.customer_id
  WHERE s.business_unit_id = $2 AND s.organization_id = $3 AND (s.created_at >= $4 AND s.created_at <= $5) AND s.status <> 'Canceled'
  GROUP BY s.customer_id, c.name
  HAVING CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE s.created_at >= $7), $8) AS double precision) > $11
)
SELECT cm.customer_id, cm.name, cm.revenue, cm.loads, cm.previous_revenue, cm.total_revenue
FROM customer_mix cm
WHERE cm.row_rank <= $1
ORDER BY cm.row_rank ASC
-- $1 = 5
-- $2 = 'bu_1'
-- $3 = 'org_1'
-- $4 = 500
-- $5 = 1500
-- $6 = ''
-- $7 = 1000
-- $8 = 0
-- $9 = 1000
-- $10 = 0
-- $11 = 0
