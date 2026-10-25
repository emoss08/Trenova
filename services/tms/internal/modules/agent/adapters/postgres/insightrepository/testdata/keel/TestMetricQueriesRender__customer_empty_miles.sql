SELECT c.id, c.name, CAST(coalesce(sum(sm.distance) FILTER (WHERE sm.loaded = $6), $7) AS text), CAST(coalesce(sum(sm.distance), $8) AS text), count(*), CAST(sum(coalesce(sum(sm.distance) FILTER (WHERE sm.loaded = $6), $7)) OVER () AS text), CAST(sum(coalesce(sum(sm.distance), $8)) OVER () AS text)
FROM shipment_moves sm
JOIN shipments s ON s.business_unit_id = $1 AND s.organization_id = $2 AND s.id = sm.shipment_id
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s.customer_id
WHERE sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.status = 'Completed' AND sm.distance IS NOT NULL AND sm.distance > $3 AND s.actual_delivery_date >= $4 AND s.actual_delivery_date < $5
GROUP BY c.id, c.name
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1790000000
-- $5 = 1792592000
-- $6 = false
-- $7 = 0
-- $8 = 0
