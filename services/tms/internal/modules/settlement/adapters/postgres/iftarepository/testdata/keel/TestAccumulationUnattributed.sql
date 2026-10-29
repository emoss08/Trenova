WITH completion_candidates (id) AS ((
  SELECT done_stop.shipment_move_id
  FROM stops done_stop
  WHERE done_stop.business_unit_id = $2 AND done_stop.organization_id = $3 AND coalesce(done_stop.actual_departure, done_stop.actual_arrival) >= $5 AND coalesce(done_stop.actual_departure, done_stop.actual_arrival) < $6
) UNION (
  SELECT done_move.id
  FROM shipment_moves done_move
  WHERE done_move.business_unit_id = $2 AND done_move.organization_id = $3 AND done_move.status = 'Completed' AND done_move.updated_at >= $7 AND done_move.updated_at < $8
)), inp (id, loaded, distance, completed_at) AS (
  SELECT sm.id, sm.loaded, CASE WHEN sm.distance_units = $9 THEN (sm.distance / $10::double precision) ELSE sm.distance END, coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at)
  FROM shipment_moves sm
  JOIN completion_candidates cc ON cc.id = sm.id
  LEFT JOIN stops s ON s.business_unit_id = $2 AND s.organization_id = $3 AND s.shipment_move_id = sm.id
  WHERE sm.business_unit_id = $2 AND sm.organization_id = $3 AND sm.status = 'Completed'
  GROUP BY sm.id, sm.loaded, sm.distance, sm.distance_units, sm.updated_at
  HAVING coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) >= $11 AND coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) < $12
)
SELECT i.id, round(CAST(i.distance AS numeric), $4)
FROM inp i
WHERE i.distance > $1 AND NOT EXISTS (
  SELECT 1
  FROM shipment_move_jurisdiction_miles smjm
  WHERE smjm.business_unit_id = $2 AND smjm.organization_id = $3 AND smjm.shipment_move_id = i.id
) AND NOT EXISTS (
  SELECT 1
  FROM ifta_jurisdiction_mileage_entries ijme
  WHERE ijme.business_unit_id = $2 AND ijme.organization_id = $3 AND ijme.shipment_move_id = i.id
)
ORDER BY i.completed_at ASC, i.id ASC
-- $1 = 0
-- $2 = 'bu_1'
-- $3 = 'org_1'
-- $4 = 3
-- $5 = 100
-- $6 = 200
-- $7 = 100
-- $8 = 200
-- $9 = 'Kilometers'
-- $10 = 1.609344
-- $11 = 100
-- $12 = 200
