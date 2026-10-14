DROP INDEX IF EXISTS "idx_agent_memories_offers";

--bun:split
UPDATE "resource_permissions"
SET "operations" = array_cat(array_remove("operations", 'manage'), ARRAY['create', 'update']::text[]),
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE "resource" = 'agent_memory'
  AND 'manage' = ANY("operations");
