-- Six resources were declared in the permission registry without a route,
-- resolver, service or tool that ever checked them, so a grant of any of them
-- allowed nothing and only made the role editor longer. They are gone from the
-- registry; the grants that named them go with them.
DELETE FROM "resource_permissions"
WHERE "resource" IN (
    'comment_type',
    'customer_contact',
    'delay_code',
    'document_classification',
    'reason_code',
    'tag'
);

--bun:split
DELETE FROM "api_key_permissions"
WHERE "resource" IN (
    'comment_type',
    'customer_contact',
    'delay_code',
    'document_classification',
    'reason_code',
    'tag'
);
