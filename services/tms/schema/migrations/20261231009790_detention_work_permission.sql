-- Working detention (the desk, its occurrences, notices, disputes and the
-- intelligence over them) is now its own resource, apart from configuring
-- detention policies. Every role and API key that could read or act on
-- detention through detention_policy keeps exactly that on detention, so
-- nothing changes on upgrade; an administrator then takes policy access away
-- from the people who only work the desk.
INSERT INTO "resource_permissions"("id", "role_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
       rp."role_id",
       'detention',
       ARRAY(
           SELECT op
           FROM unnest(ARRAY['read', 'update']::text[]) AS op
           WHERE op = ANY(rp."operations")
       ),
       rp."data_scope",
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "resource_permissions" rp
WHERE rp."resource" = 'detention_policy'
  AND rp."operations" && ARRAY['read', 'update']::text[]
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" existing
      WHERE existing."role_id" = rp."role_id"
        AND existing."resource" = 'detention'
  );

INSERT INTO "api_key_permissions"("id", "api_key_id", "business_unit_id", "organization_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('akp_', replace(gen_random_uuid()::text, '-', '')),
       akp."api_key_id",
       akp."business_unit_id",
       akp."organization_id",
       'detention',
       ARRAY(
           SELECT op
           FROM unnest(ARRAY['read', 'update']::text[]) AS op
           WHERE op = ANY(akp."operations")
       ),
       akp."data_scope",
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "api_key_permissions" akp
WHERE akp."resource" = 'detention_policy'
  AND akp."operations" && ARRAY['read', 'update']::text[]
  AND NOT EXISTS (
      SELECT 1
      FROM "api_key_permissions" existing
      WHERE existing."api_key_id" = akp."api_key_id"
        AND existing."resource" = 'detention'
  );
