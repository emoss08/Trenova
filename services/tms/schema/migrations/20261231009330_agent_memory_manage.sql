-- Keeping, changing, retiring and approving what agents remember for a team,
-- the organization or every agent is one permission, agent_memory manage,
-- which the organization's administrators hold. The create and update
-- operations it replaces no longer exist: a role that held them offers shared
-- memory for an administrator to keep, as everyone does.
UPDATE "resource_permissions" rp
SET "operations" = array_append(array_remove(array_remove(rp."operations", 'create'), 'update'), 'manage'),
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "roles" r
WHERE rp."role_id" = r."id"
  AND r."is_system" = true
  AND r."name" = 'Organization Administrator'
  AND rp."resource" = 'agent_memory'
  AND NOT ('manage' = ANY(rp."operations"));

INSERT INTO "resource_permissions"("id", "role_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
       r."id",
       'agent_memory',
       ARRAY['read', 'manage']::text[],
       'organization',
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "roles" r
WHERE r."is_system" = true
  AND r."name" = 'Organization Administrator'
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" rp
      WHERE rp."role_id" = r."id"
        AND rp."resource" = 'agent_memory'
  );

DELETE FROM "resource_permissions"
WHERE "resource" = 'agent_memory'
  AND cardinality(array_remove(array_remove("operations", 'create'), 'update')) = 0;

UPDATE "resource_permissions"
SET "operations" = array_remove(array_remove("operations", 'create'), 'update'),
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE "resource" = 'agent_memory'
  AND ('create' = ANY("operations") OR 'update' = ANY("operations"));

-- The Desk lists a person's own offers beside what is kept for them.
CREATE INDEX IF NOT EXISTS "idx_agent_memories_offers"
    ON "agent_memories"("organization_id", "business_unit_id", "created_by_user_id")
    WHERE "status" = 'Suggested' AND "source" = 'User';
