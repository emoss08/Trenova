SELECT cs.id AS c1
FROM check_states cs
WHERE (cs.id = $1) LIMIT 1
-- $1 = 'st_1'
