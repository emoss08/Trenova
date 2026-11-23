-- AI usage and spend is a resource of its own (ai_usage) rather than a read of AI providers. A
-- role that could read providers saw their usage, so every role that holds any operation on
-- ai_provider keeps seeing usage: it is granted ai_usage read, and export where it exports
-- providers or reports. Administrator roles are completed by the administrator_permissions
-- seed, like any new resource.
INSERT INTO "resource_permissions" ("id", "role_id", "resource", "operations", "data_scope")
SELECT DISTINCT ON (p."role_id")
    'rp_' || substr(md5(p."role_id" || ':ai_usage'), 1, 26),
    p."role_id",
    'ai_usage',
    CASE
        WHEN 'export' = ANY(p."operations") OR EXISTS (
            SELECT 1
            FROM "resource_permissions" r
            WHERE r."role_id" = p."role_id"
              AND r."resource" = 'report'
              AND 'export' = ANY(r."operations")
        ) THEN ARRAY['read', 'export']::text[]
        ELSE ARRAY['read']::text[]
    END,
    'organization'
FROM "resource_permissions" p
WHERE p."resource" = 'ai_provider'
  AND cardinality(p."operations") > 0
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" existing
      WHERE existing."role_id" = p."role_id"
        AND existing."resource" = 'ai_usage'
  )
ORDER BY p."role_id", ('export' = ANY(p."operations")) DESC;
