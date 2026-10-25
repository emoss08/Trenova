SELECT DISTINCT ON (to_char(ds.day, $6), wp.type, wp.worker_id) to_char(ds.day, $6), wp.type, wp.worker_id, w.first_name, w.last_name
FROM worker_pto wp
JOIN workers w ON w.business_unit_id = $1 AND w.organization_id = $2 AND w.id = wp.worker_id
CROSS JOIN LATERAL (
  SELECT generate_series(greatest(date_trunc('day', timezone($7::text, to_timestamp(wp.start_date))), date_trunc('day', timezone($7::text, to_timestamp($8::bigint)))), least(date_trunc('day', timezone($7::text, to_timestamp(wp.end_date))), date_trunc('day', timezone($7::text, to_timestamp($9::bigint)))), $10::interval) AS day
) ds
WHERE wp.business_unit_id = $1 AND wp.organization_id = $2 AND wp.status = 'Approved' AND wp.end_date >= $3 AND wp.start_date <= $4 AND wp.type = 'Vacation' AND wp.worker_id = $5
ORDER BY to_char(ds.day, $6) ASC, wp.type ASC, wp.worker_id ASC
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 90
-- $4 = 210
-- $5 = 'wrk_1'
-- $6 = 'YYYY-MM-DD'
-- $7 = 'America/Chicago'
-- $8 = 100
-- $9 = 200
-- $10 = '1 day'
