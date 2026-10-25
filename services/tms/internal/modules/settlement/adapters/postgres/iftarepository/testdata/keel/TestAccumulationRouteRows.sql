WITH inp (id, loaded, distance, completed_at) AS (
  SELECT sm.id, sm.loaded, CASE WHEN sm.distance_units = $9 THEN (sm.distance / $10) ELSE sm.distance END, coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at)
  FROM shipment_moves sm
  LEFT JOIN stops s ON s.business_unit_id = $7 AND s.organization_id = $8 AND s.shipment_move_id = sm.id
  WHERE sm.business_unit_id = $7 AND sm.organization_id = $8 AND sm.status = 'Completed'
  GROUP BY sm.id, sm.loaded, sm.distance, sm.distance_units, sm.updated_at
  HAVING (coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) >= $11) AND (coalesce(max(coalesce(s.actual_departure, s.actual_arrival)), sm.updated_at) < $12)
), jm (shipment_move_id, country_code, jurisdiction_code, loaded, move_distance, miles) AS (
  SELECT smjm.shipment_move_id, smjm.country_code, smjm.jurisdiction_code, i.loaded, i.distance, CASE WHEN smjm.distance_units = $13 THEN (smjm.distance / $14) ELSE smjm.distance END
  FROM shipment_move_jurisdiction_miles smjm
  JOIN inp i ON i.id = smjm.shipment_move_id
  WHERE smjm.business_unit_id = $7 AND smjm.organization_id = $8 AND NOT EXISTS (
    SELECT 1
    FROM ifta_jurisdiction_mileage_entries ijme
    WHERE ijme.business_unit_id = $7 AND ijme.organization_id = $8 AND ijme.shipment_move_id = smjm.shipment_move_id
  )
)
SELECT a.tractor_id, j.country_code, j.jurisdiction_code, ij.id, round(CAST(coalesce(sum(j.miles), $1) AS numeric), $2), round(CAST(coalesce(sum(j.miles) FILTER (WHERE (j.loaded IS TRUE)), $3) AS numeric), $4), round(CAST(coalesce(sum(j.miles) FILTER (WHERE (j.loaded IS FALSE)), $5) AS numeric), $6), count(DISTINCT j.shipment_move_id)
FROM jm j
LEFT JOIN assignments a ON a.business_unit_id = $7 AND a.organization_id = $8 AND a.shipment_move_id = j.shipment_move_id AND a.archived_at IS NULL
LEFT JOIN ifta_jurisdictions ij ON ij.country_code = j.country_code AND ij.code = j.jurisdiction_code
GROUP BY a.tractor_id, j.country_code, j.jurisdiction_code, ij.id
ORDER BY a.tractor_id ASC NULLS FIRST, j.country_code ASC, j.jurisdiction_code ASC
-- $1 = 0
-- $2 = 3
-- $3 = 0
-- $4 = 3
-- $5 = 0
-- $6 = 3
-- $7 = 'bu_1'
-- $8 = 'org_1'
-- $9 = 'Kilometers'
-- $10 = 1.609344
-- $11 = 100
-- $12 = 200
-- $13 = 'Kilometers'
-- $14 = 1.609344
