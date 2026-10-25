SELECT s2.pro_number, c.name, CAST(($7 - s.actual_arrival) AS bigint)
FROM stops s
JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.id = s.shipment_move_id
JOIN shipments s2 ON s2.business_unit_id = $1 AND s2.organization_id = $2 AND s2.id = sm.shipment_id
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s2.customer_id
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status <> 'Canceled' AND s.actual_arrival IS NOT NULL AND s.actual_arrival > $3 AND (s.actual_departure IS NULL OR s.actual_departure = $4) AND (($5 - s.actual_arrival) > $6)
ORDER BY 3 DESC, s2.pro_number ASC LIMIT 10
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 0
-- $5 = 1500
-- $6 = 7200
-- $7 = 1500
