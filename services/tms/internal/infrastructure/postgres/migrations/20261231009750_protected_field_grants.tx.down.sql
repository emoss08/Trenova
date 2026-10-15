DELETE FROM "resource_permissions"
WHERE "resource" IN ('protected_personal', 'protected_margins');

--bun:split
DELETE FROM "api_key_permissions"
WHERE "resource" IN ('protected_personal', 'protected_margins');
