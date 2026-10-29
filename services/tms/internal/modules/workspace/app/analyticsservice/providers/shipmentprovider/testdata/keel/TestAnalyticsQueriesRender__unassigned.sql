SELECT count(*), count(*) FILTER (WHERE ((s.created_at >= $3 AND s.created_at <= $4))), count(*) FILTER (WHERE ((s.created_at >= $5 AND s.created_at <= $6))), CAST(coalesce(sum(s.total_charge_amount), $7) AS double precision)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted') AND s.has_active_assignment IS FALSE
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 0
-- $6 = 500
-- $7 = 0
