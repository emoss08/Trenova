UPDATE "resource_permissions"
SET "operations" = array_remove("operations", 'post'),
    "updated_at" = EXTRACT(EPOCH FROM current_timestamp)::bigint
WHERE "resource" = 'invoice'
  AND 'post' = ANY("operations");
