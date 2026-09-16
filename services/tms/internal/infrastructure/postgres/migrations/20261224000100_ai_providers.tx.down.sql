DROP INDEX IF EXISTS "idx_ai_providers_tasks";

DROP INDEX IF EXISTS "idx_ai_providers_routing";

DROP INDEX IF EXISTS "uq_ai_providers_org_name";

--bun:split
DROP TABLE IF EXISTS "ai_providers";
