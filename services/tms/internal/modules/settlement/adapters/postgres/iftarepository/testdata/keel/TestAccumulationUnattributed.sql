WITH inp (id, loaded, distance, completed_at) AS (
  SELECT sm.id, sm.loaded, CASE WHEN sm.distance_units = $5 THEN (sm.distance / $6) ELSE sm.distance END, coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at)
  FROM shipment_moves sm
  LEFT JOIN stops s ON s.business_unit_id = $2 AND s.organization_id = $3 AND s.shipment_move_id = sm.id
  WHERE sm.business_unit_id = $2 AND sm.organization_id = $3 AND sm.status = 'Completed'
  GROUP BY sm.id, sm.loaded, sm.distance, sm.distance_units, sm.updated_at
  HAVING coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) >= $7 AND coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) < $8
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
-- $5 = 'Kilometers'
-- $6 = 1.609344
-- $7 = 100
-- $8 = 200
