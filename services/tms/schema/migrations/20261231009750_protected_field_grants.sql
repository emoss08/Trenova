-- A worker's date of birth, licence, TWIC and credential numbers and drug test
-- dates, and a rate agreement's margin floor and pay ceiling, are now withheld
-- from anyone whose role does not grant protected_personal or
-- protected_margins. Every role that could read those records before keeps
-- seeing them, so nothing disappears on upgrade; an administrator narrows the
-- roles that should not. The Organization Administrator role gets both from
-- ReconcileAdminRoles after the migrations, and other system roles are left alone.
INSERT INTO "resource_permissions"("id", "role_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
       r."id",
       'protected_personal',
       ARRAY['read']::text[],
       'organization',
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "roles" r
WHERE r."is_system" = false
  AND EXISTS (
      SELECT 1
      FROM "resource_permissions" rp
      WHERE rp."role_id" = r."id"
        AND rp."resource" IN ('worker', 'worker_credential')
        AND 'read' = ANY(rp."operations")
  )
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" rp
      WHERE rp."role_id" = r."id"
        AND rp."resource" = 'protected_personal'
  );

INSERT INTO "resource_permissions"("id", "role_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
       r."id",
       'protected_margins',
       ARRAY['read']::text[],
       'organization',
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "roles" r
WHERE r."is_system" = false
  AND EXISTS (
      SELECT 1
      FROM "resource_permissions" rp
      WHERE rp."role_id" = r."id"
        AND rp."resource" = 'rate_agreement'
        AND 'read' = ANY(rp."operations")
  )
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" rp
      WHERE rp."role_id" = r."id"
        AND rp."resource" = 'protected_margins'
  );

INSERT INTO "api_key_permissions"("id", "api_key_id", "business_unit_id", "organization_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('akp_', replace(gen_random_uuid()::text, '-', '')),
       akp."api_key_id",
       akp."business_unit_id",
       akp."organization_id",
       'protected_personal',
       ARRAY['read']::text[],
       'organization',
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "api_key_permissions" akp
WHERE akp."resource" = 'worker'
  AND 'read' = ANY(akp."operations")
  AND NOT EXISTS (
      SELECT 1
      FROM "api_key_permissions" existing
      WHERE existing."api_key_id" = akp."api_key_id"
        AND existing."resource" = 'protected_personal'
  );
