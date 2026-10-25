SELECT s2.id, s2.pro_number, s.scheduled_window_start, coalesce(c.name, $5), coalesce(nullif(pickup_loc.code, $6), coalesce(nullif(pickup_loc.city, $7), coalesce(pickup_loc.name, $8))), coalesce(nullif(dest_loc.code, $6), coalesce(nullif(dest_loc.city, $7), coalesce(dest_loc.name, $8))), coalesce(nullif(concat_ws($9::text, nullif(wrk.first_name, $10), nullif(wrk.last_name, $11)), $12), $13), s2.status, a.primary_worker_id IS NOT NULL
FROM stops s
JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.id = s.shipment_move_id
JOIN shipments s2 ON s2.business_unit_id = $1 AND s2.organization_id = $2 AND s2.id = sm.shipment_id
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s2.customer_id
JOIN locations pickup_loc ON pickup_loc.business_unit_id = $1 AND pickup_loc.organization_id = $2 AND pickup_loc.id = s.location_id
LEFT JOIN assignments a ON a.business_unit_id = $1 AND a.organization_id = $2 AND a.shipment_move_id = sm.id AND a.archived_at IS NULL AND a.status <> 'Canceled'
LEFT JOIN workers wrk ON wrk.business_unit_id = $1 AND wrk.organization_id = $2 AND wrk.id = a.primary_worker_id
LEFT JOIN LATERAL (
  SELECT dest_loc2.code, dest_loc2.city, dest_loc2.name
  FROM locations dest_loc2
  JOIN stops stp_dest ON stp_dest.business_unit_id = $1 AND stp_dest.organization_id = $2 AND stp_dest.location_id = dest_loc2.id
  JOIN shipment_moves sm_dest ON sm_dest.business_unit_id = $1 AND sm_dest.organization_id = $2 AND sm_dest.id = stp_dest.shipment_move_id
  WHERE dest_loc2.business_unit_id = $1 AND dest_loc2.organization_id = $2 AND sm_dest.shipment_id = s2.id AND stp_dest.status <> 'Canceled' AND stp_dest.type IN ('Delivery', 'SplitDelivery')
  ORDER BY sm_dest.sequence DESC, stp_dest.sequence DESC LIMIT 1
) dest_loc ON TRUE
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.type IN ('Pickup', 'SplitPickup') AND s.status <> 'Canceled' AND sm.status <> 'Canceled' AND s2.status <> 'Canceled' AND s.scheduled_window_start >= $3 AND s.scheduled_window_start < $4
ORDER BY s.scheduled_window_start ASC, s2.pro_number ASC LIMIT 20
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 2000
-- $4 = 3000
-- $5 = ''
-- $6 = ''
-- $7 = ''
-- $8 = ''
-- $9 = ' '
-- $10 = ''
-- $11 = ''
-- $12 = ''
-- $13 = ''
