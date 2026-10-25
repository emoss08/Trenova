SELECT CAST(date_part('hour', timezone($5::text, to_timestamp(s.actual_delivery_date))) AS bigint), CAST(coalesce(sum(s.total_charge_amount), $6) AS double precision)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND (s.actual_delivery_date >= $3 AND s.actual_delivery_date <= $4)
GROUP BY CAST(date_part('hour', timezone($5::text, to_timestamp(s.actual_delivery_date))) AS bigint)
ORDER BY CAST(date_part('hour', timezone($5::text, to_timestamp(s.actual_delivery_date))) AS bigint) ASC
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 'America/Chicago'
-- $6 = 0
