SELECT count(*), count(*) FILTER (WHERE ((s.created_at >= $3 AND s.created_at <= $4))), count(*) FILTER (WHERE ((s.created_at >= $5 AND s.created_at <= $6))), CAST(coalesce(sum(s.total_charge_amount), $7) AS double precision)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted') AND NOT EXISTS (
  SELECT 1
  FROM shipment_moves sm
  JOIN assignments a ON a.business_unit_id = $1 AND a.organization_id = $2 AND a.shipment_move_id = sm.id AND a.archived_at IS NULL AND a.status <> 'Canceled'
  WHERE sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.shipment_id = s.id AND sm.status <> 'Canceled'
)
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 0
-- $6 = 500
-- $7 = 0
