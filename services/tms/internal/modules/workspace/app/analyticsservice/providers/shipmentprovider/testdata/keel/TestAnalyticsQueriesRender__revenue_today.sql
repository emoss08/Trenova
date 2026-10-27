SELECT CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE (s.actual_delivery_date >= $5 AND s.actual_delivery_date <= $4)), $6) AS double precision), CAST(coalesce(sum(s.total_charge_amount) FILTER (WHERE (s.actual_delivery_date >= $3 AND s.actual_delivery_date <= $7)), $8) AS double precision), (
  SELECT CAST(coalesce(sum(sm.distance), $10) AS double precision)
  FROM shipment_moves sm
  JOIN shipments mi_sp ON mi_sp.business_unit_id = $1 AND mi_sp.organization_id = $2 AND mi_sp.id = sm.shipment_id
  WHERE sm.business_unit_id = $1 AND sm.organization_id = $2 AND (mi_sp.actual_delivery_date >= $5 AND mi_sp.actual_delivery_date <= $4) AND sm.distance IS NOT NULL AND sm.distance > $9
)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND (s.actual_delivery_date >= $3 AND s.actual_delivery_date <= $4)
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1500
-- $5 = 1000
-- $6 = 0
-- $7 = 500
-- $8 = 0
-- $9 = 0
-- $10 = 0
