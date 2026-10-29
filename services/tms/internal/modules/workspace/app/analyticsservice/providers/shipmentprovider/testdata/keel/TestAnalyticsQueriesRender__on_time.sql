SELECT count(*) FILTER (WHERE ((s.actual_arrival >= $6 AND s.actual_arrival <= $4))), count(*) FILTER (WHERE ((s.actual_arrival >= $6 AND s.actual_arrival <= $4) AND s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start))), count(*) FILTER (WHERE ((s.actual_arrival >= $7 AND s.actual_arrival <= $8))), count(*) FILTER (WHERE ((s.actual_arrival >= $7 AND s.actual_arrival <= $8) AND s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start))), count(*) FILTER (WHERE ((s.actual_arrival >= $3 AND s.actual_arrival < $9))), count(*) FILTER (WHERE ((s.actual_arrival >= $3 AND s.actual_arrival < $9) AND s.actual_arrival <= coalesce(s.scheduled_window_end, s.scheduled_window_start)))
FROM stops s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.status = 'Completed' AND s.type IN ('Delivery', 'SplitDelivery') AND (s.actual_arrival >= $3 AND s.actual_arrival <= $4) AND s.scheduled_window_start > $5
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = -6000
-- $4 = 1500
-- $5 = 0
-- $6 = 1000
-- $7 = 0
-- $8 = 500
-- $9 = 1000
