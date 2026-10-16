DELETE FROM "resource_permissions"
WHERE "resource" = 'detention';

--bun:split
DELETE FROM "api_key_permissions"
WHERE "resource" = 'detention';
