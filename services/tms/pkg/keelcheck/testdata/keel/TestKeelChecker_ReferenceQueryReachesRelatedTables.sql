SELECT ck.id AS c1
FROM check_kinds ck
WHERE ck.organization_id = $1 AND ck.business_unit_id = $2 AND (ck.id = $3) LIMIT 1
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 'ck_1'
