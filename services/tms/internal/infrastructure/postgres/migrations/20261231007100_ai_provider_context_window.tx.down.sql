ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "chk_ai_providers_context_window";

ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "context_window_tokens";
