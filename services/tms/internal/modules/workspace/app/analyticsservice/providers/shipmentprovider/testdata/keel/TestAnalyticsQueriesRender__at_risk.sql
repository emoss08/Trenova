SELECT count(*), count(*) FILTER (WHERE ((s.created_at >= $5 AND s.created_at <= $6))), count(*) FILTER (WHERE ((s.created_at >= $7 AND s.created_at <= $8))), count(*) FILTER (WHERE (EXISTS (
  SELECT 1
  FROM stop_etas se2
  WHERE se2.organization_id = $2 AND se2.business_unit_id = $1 AND se2.shipment_id = s.id AND se2.verdict IN ('AtRisk', 'Late') AND se2.arrived IS FALSE
))), (
  SELECT count(*)
  FROM weather_alerts wa
  WHERE wa.organization_id = $2 AND wa.business_unit_id = $1 AND wa.expired_at IS NULL AND (wa.expires IS NULL OR wa.expires >= $9)
), count(*) FILTER (WHERE ((s.temperature_min IS NOT NULL OR s.temperature_max IS NOT NULL)))
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted') AND (s.status = 'Delayed' OR EXISTS (
  SELECT 1
  FROM shipment_moves m
  WHERE m.business_unit_id = $1 AND m.organization_id = $2 AND m.shipment_id = s.id AND EXISTS (
    SELECT 1
    FROM stops s2
    WHERE s2.business_unit_id = $1 AND s2.organization_id = $2 AND s2.shipment_move_id = m.id AND s2.status IN ('New', 'InTransit') AND s2.scheduled_window_start > $3 AND s2.scheduled_window_start < $4
  )
) OR EXISTS (
  SELECT 1
  FROM stop_etas se
  WHERE se.organization_id = $2 AND se.business_unit_id = $1 AND se.shipment_id = s.id AND se.verdict IN ('AtRisk', 'Late') AND se.arrived IS FALSE
))
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1500
-- $5 = 1000
-- $6 = 1500
-- $7 = 0
-- $8 = 500
-- $9 = 1500
