SELECT c.id, c.name, count(*), CAST(coalesce(sum(s.total_charge_amount), $5) AS text), min(s.actual_delivery_date)
FROM shipments s
JOIN customers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = s.customer_id
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.actual_delivery_date IS NOT NULL AND s.actual_delivery_date > $3 AND s.actual_delivery_date < $4 AND s.billed_at IS NULL AND s.status IN ('Completed', 'ReadyToInvoice')
GROUP BY c.id, c.name
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1790000000
-- $5 = 0
