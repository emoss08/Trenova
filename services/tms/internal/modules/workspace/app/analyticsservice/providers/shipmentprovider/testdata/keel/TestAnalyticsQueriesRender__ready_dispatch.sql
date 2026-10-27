WITH shipment_assignments (id, primary_worker_id) AS (
  SELECT DISTINCT ON (ra_sp.id, a.primary_worker_id) ra_sp.id, a.primary_worker_id
  FROM shipments ra_sp
  JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.shipment_id = ra_sp.id AND sm.status <> 'Canceled'
  JOIN assignments a ON a.business_unit_id = $1 AND a.organization_id = $2 AND a.shipment_move_id = sm.id AND a.archived_at IS NULL AND a.status <> 'Canceled'
  WHERE ra_sp.business_unit_id = $1 AND ra_sp.organization_id = $2 AND ra_sp.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted')
  ORDER BY ra_sp.id ASC, a.primary_worker_id ASC
)
SELECT count(*) FILTER (WHERE ((s.status = 'Assigned' AND sa.id IS NOT NULL))), count(*) FILTER (WHERE ((s.status = 'Assigned' AND sa.id IS NOT NULL) AND (s.created_at >= $3 AND s.created_at <= $4))), count(*) FILTER (WHERE ((s.status = 'Assigned' AND sa.id IS NOT NULL) AND s.created_at >= $5 AND s.created_at < $6)), count(*) FILTER (WHERE (sa.id IS NULL)), count(*) FILTER (WHERE (sa.primary_worker_id IS NOT NULL))
FROM shipments s
LEFT JOIN shipment_assignments sa ON sa.id = s.id
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted')
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 0
-- $6 = 500
