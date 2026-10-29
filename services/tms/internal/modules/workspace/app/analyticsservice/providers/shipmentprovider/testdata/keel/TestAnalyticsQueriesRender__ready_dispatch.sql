SELECT count(*) FILTER (WHERE ((s.status = 'Assigned' AND s.has_active_assignment IS TRUE))), count(*) FILTER (WHERE ((s.status = 'Assigned' AND s.has_active_assignment IS TRUE) AND (s.created_at >= $3 AND s.created_at <= $4))), count(*) FILTER (WHERE ((s.status = 'Assigned' AND s.has_active_assignment IS TRUE) AND s.created_at >= $5 AND s.created_at < $6)), count(*) FILTER (WHERE (s.has_active_assignment IS FALSE)), count(*) FILTER (WHERE (drv_a.id IS NOT NULL))
FROM shipments s
LEFT JOIN LATERAL (
  SELECT drv_a2.id
  FROM assignments drv_a2
  JOIN shipment_moves drv_sm ON drv_sm.business_unit_id = $1 AND drv_sm.organization_id = $2 AND drv_sm.id = drv_a2.shipment_move_id
  WHERE drv_a2.business_unit_id = $1 AND drv_a2.organization_id = $2 AND s.has_active_assignment IS TRUE AND drv_sm.shipment_id = s.id AND drv_sm.status <> 'Canceled' AND drv_a2.archived_at IS NULL AND drv_a2.status <> 'Canceled' AND drv_a2.primary_worker_id IS NOT NULL LIMIT 1
) drv_a ON TRUE
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted')
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 0
-- $6 = 500
