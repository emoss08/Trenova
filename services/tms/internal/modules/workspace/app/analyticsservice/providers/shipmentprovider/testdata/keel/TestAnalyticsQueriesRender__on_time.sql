SELECT count(*) FILTER (WHERE ((s.actual_arrival >= $5 AND s.actual_arrival <= $6))), count(*) FILTER (WHERE ((s.actual_arrival >= $5 AND s.actual_arrival <= $6) AND (s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start)))), count(*) FILTER (WHERE ((s.actual_arrival >= $7 AND s.actual_arrival <= $8))), count(*) FILTER (WHERE ((s.actual_arrival >= $7 AND s.actual_arrival <= $8) AND (s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start)))), count(*) FILTER (WHERE ((s.actual_arrival >= $9 AND s.actual_arrival < $10))), count(*) FILTER (WHERE ((s.actual_arrival >= $9 AND s.actual_arrival < $10) AND (s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start))))
FROM stops s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status = 'Completed' AND s.type IN ('Delivery', 'SplitDelivery') AND s.actual_arrival IS NOT NULL AND s.actual_arrival > $3 AND s.scheduled_window_start > $4
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 0
-- $5 = 1000
-- $6 = 1500
-- $7 = 0
-- $8 = 500
-- $9 = -6000
-- $10 = 1000
