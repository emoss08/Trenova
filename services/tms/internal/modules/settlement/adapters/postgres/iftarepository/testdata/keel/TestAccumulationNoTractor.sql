WITH inp (id, loaded, distance, completed_at) AS (
  SELECT sm.id, sm.loaded, CASE WHEN sm.distance_units = $5 THEN (sm.distance / $6) ELSE sm.distance END, coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at)
  FROM shipment_moves sm
  LEFT JOIN stops s ON s.business_unit_id = $3 AND s.organization_id = $4 AND s.shipment_move_id = sm.id
  WHERE sm.business_unit_id = $3 AND sm.organization_id = $4 AND sm.status = 'Completed'
  GROUP BY sm.id, sm.loaded, sm.distance, sm.distance_units, sm.updated_at
  HAVING (coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) >= $7) AND (coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) < $8)
), jm (shipment_move_id, country_code, jurisdiction_code, loaded, move_distance, miles) AS (
  SELECT smjm.shipment_move_id, smjm.country_code, smjm.jurisdiction_code, i.loaded, i.distance, CASE WHEN smjm.distance_units = $9 THEN (smjm.distance / $10) ELSE smjm.distance END
  FROM shipment_move_jurisdiction_miles smjm
  JOIN inp i ON i.id = smjm.shipment_move_id
  WHERE smjm.business_unit_id = $3 AND smjm.organization_id = $4 AND NOT EXISTS (
    SELECT 1
    FROM ifta_jurisdiction_mileage_entries ijme
    WHERE ijme.business_unit_id = $3 AND ijme.organization_id = $4 AND ijme.shipment_move_id = smjm.shipment_move_id
  )
)
SELECT j.shipment_move_id, round(CAST(coalesce(sum(j.miles), $1) AS numeric), $2)
FROM jm j
LEFT JOIN assignments a ON a.business_unit_id = $3 AND a.organization_id = $4 AND a.shipment_move_id = j.shipment_move_id AND a.archived_at IS NULL
WHERE a.tractor_id IS NULL
GROUP BY j.shipment_move_id
ORDER BY j.shipment_move_id ASC
-- $1 = 0
-- $2 = 3
-- $3 = 'bu_1'
-- $4 = 'org_1'
-- $5 = 'Kilometers'
-- $6 = 1.609344
-- $7 = 100
-- $8 = 200
-- $9 = 'Kilometers'
-- $10 = 1.609344
