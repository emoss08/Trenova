SELECT ci.id AS c1
FROM check_items ci
WHERE ci.organization_id = $1 AND ci.business_unit_id = $2 AND (lower(ci.code) = lower($3::text)) AND (ci.kind_id = $4) AND (ci.id <> $5) LIMIT 1
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 'ABC'
-- $4 = 'ck_1'
-- $5 = 'ci_1'
