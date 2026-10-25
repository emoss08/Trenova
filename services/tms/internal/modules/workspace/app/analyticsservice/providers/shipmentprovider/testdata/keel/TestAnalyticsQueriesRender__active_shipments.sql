SELECT count(*) FILTER (WHERE (s.status IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted'))), count(*) FILTER (WHERE ((s.created_at >= $3 AND s.created_at <= $4))), count(*) FILTER (WHERE ((s.created_at >= $5 AND s.created_at <= $6))), count(*) FILTER (WHERE (s.status = 'InTransit')), count(*) FILTER (WHERE (s.status = 'Delayed')), count(*) FILTER (WHERE (s.status = 'Assigned')), count(*) FILTER (WHERE (s.status IN ('Completed', 'Invoiced', 'ReadyToInvoice')))
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1000
-- $4 = 1500
-- $5 = 0
-- $6 = 500
