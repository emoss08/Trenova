ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_context_window";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "context_window";
