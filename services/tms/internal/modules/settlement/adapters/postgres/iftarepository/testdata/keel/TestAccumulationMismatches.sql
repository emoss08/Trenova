WITH completion_candidates (id) AS ((
  SELECT done_stop.shipment_move_id
  FROM stops done_stop
  WHERE done_stop.business_unit_id = $5 AND done_stop.organization_id = $6 AND coalesce(done_stop.actual_departure, done_stop.actual_arrival) >= $7 AND coalesce(done_stop.actual_departure, done_stop.actual_arrival) < $8
) UNION (
  SELECT done_move.id
  FROM shipment_moves done_move
  WHERE done_move.business_unit_id = $5 AND done_move.organization_id = $6 AND done_move.status = 'Completed' AND done_move.updated_at >= $9 AND done_move.updated_at < $10
)), inp (id, loaded, distance, completed_at) AS (
  SELECT sm.id, sm.loaded, CASE WHEN sm.distance_units = $11 THEN (sm.distance / $12::double precision) ELSE sm.distance END, coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at)
  FROM shipment_moves sm
  JOIN completion_candidates cc ON cc.id = sm.id
  LEFT JOIN stops s ON s.business_unit_id = $5 AND s.organization_id = $6 AND s.shipment_move_id = sm.id
  WHERE sm.business_unit_id = $5 AND sm.organization_id = $6 AND sm.status = 'Completed'
  GROUP BY sm.id, sm.loaded, sm.distance, sm.distance_units, sm.updated_at
  HAVING coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) >= $13 AND coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) < $14
), jm (shipment_move_id, country_code, jurisdiction_code, loaded, move_distance, miles) AS (
  SELECT smjm.shipment_move_id, smjm.country_code, smjm.jurisdiction_code, i.loaded, i.distance, CASE WHEN smjm.distance_units = $15 THEN (smjm.distance / $12::double precision) ELSE smjm.distance END
  FROM shipment_move_jurisdiction_miles smjm
  JOIN inp i ON i.id = smjm.shipment_move_id
  WHERE smjm.business_unit_id = $5 AND smjm.organization_id = $6 AND NOT EXISTS (
    SELECT 1
    FROM ifta_jurisdiction_mileage_entries ijme
    WHERE ijme.business_unit_id = $5 AND ijme.organization_id = $6 AND ijme.shipment_move_id = smjm.shipment_move_id
  )
)
SELECT j.shipment_move_id, round(CAST(j.move_distance AS numeric), $1), round(CAST(sum(j.miles) AS numeric), $2)
FROM jm j
WHERE j.move_distance IS NOT NULL
GROUP BY j.shipment_move_id, j.move_distance
HAVING abs((sum(j.miles) - j.move_distance)) > greatest($3::double precision, (j.move_distance * $4::double precision))
ORDER BY j.shipment_move_id ASC
-- $1 = 3
-- $2 = 3
-- $3 = 1
-- $4 = 0.02
-- $5 = 'bu_1'
-- $6 = 'org_1'
-- $7 = 100
-- $8 = 200
-- $9 = 100
-- $10 = 200
-- $11 = 'Kilometers'
-- $12 = 1.609344
-- $13 = 100
-- $14 = 200
-- $15 = 'Kilometers'
