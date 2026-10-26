SELECT c.id, c.name, count(*) FILTER (WHERE (s.actual_arrival >= $7 AND s.actual_arrival < $6)), count(*) FILTER (WHERE ((s.actual_arrival >= $7 AND s.actual_arrival < $6) AND s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start))), count(*) FILTER (WHERE (s.actual_arrival >= $5 AND s.actual_arrival < $8)), count(*) FILTER (WHERE ((s.actual_arrival >= $5 AND s.actual_arrival < $8) AND s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start)))
FROM stops s
JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.id = s.shipment_move_id
JOIN shipments s2 ON s2.business_unit_id = $1 AND s2.organization_id = $2 AND s2.id = sm.shipment_id
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s2.customer_id
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status = 'Completed' AND s.type IN ('Delivery', 'SplitDelivery') AND s.actual_arrival IS NOT NULL AND s.actual_arrival > $3 AND s.scheduled_window_start > $4 AND s.actual_arrival >= $5 AND s.actual_arrival < $6
GROUP BY c.id, c.name
HAVING count(*) FILTER (WHERE (s.actual_arrival >= $7 AND s.actual_arrival < $6)) >= $9
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 0
-- $5 = 1787408000
-- $6 = 1792592000
-- $7 = 1790000000
-- $8 = 1790000000
-- $9 = 12
