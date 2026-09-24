ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_embedding_task";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_embedding_input_style";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_embedding_dimensions";

--bun:split
UPDATE "ai_providers"
SET "tasks" = array_remove("tasks", 'Embedding')
WHERE 'Embedding' = ANY ("tasks");

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "embedding_input_style";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "embedding_dimensions";
