SELECT ci.id AS c1
FROM check_items ci
WHERE (ci.organization_id = $1) AND (ci.business_unit_id = $2) AND (ci.id = $3) LIMIT 1
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 'ci_1'
