SELECT s.id, coalesce(s.pro_number, $4), s.status, s.customer_id, coalesce(s.bill_to_customer_id, s.customer_id), coalesce(nxt.id, $5), nxt.scheduled_window_start, ARRAY(
  SELECT DISTINCT ca.carrier_id
  FROM carrier_assignments ca
  JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.id = ca.shipment_move_id
  WHERE ca.business_unit_id = $1 AND ca.organization_id = $2 AND sm.shipment_id = s.id AND ca.status <> 'Canceled'
)
FROM shipments s
LEFT JOIN LATERAL (
  SELECT nxt2.id, nxt2.scheduled_window_start
  FROM stops nxt2
  JOIN shipment_moves nmv ON nmv.business_unit_id = $1 AND nmv.organization_id = $2 AND nmv.id = nxt2.shipment_move_id
  WHERE nxt2.business_unit_id = $1 AND nxt2.organization_id = $2 AND nmv.shipment_id = s.id AND nmv.status <> 'Canceled' AND nxt2.status <> 'Canceled' AND nxt2.actual_arrival IS NULL
  ORDER BY nmv.sequence ASC, nxt2.sequence ASC LIMIT 1
) nxt ON TRUE
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.id = ANY($3)
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = '{shp_1,shp_2}'
-- $4 = ''
-- $5 = ''
