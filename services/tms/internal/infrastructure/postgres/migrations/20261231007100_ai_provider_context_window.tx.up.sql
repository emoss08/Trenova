ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "context_window_tokens" integer;

ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "chk_ai_providers_context_window";

ALTER TABLE "ai_providers"
    ADD CONSTRAINT "chk_ai_providers_context_window"
    CHECK ("context_window_tokens" IS NULL OR "context_window_tokens" BETWEEN 2048 AND 2000000);

COMMENT ON COLUMN "ai_providers"."context_window_tokens" IS 'Tokens the model reads in one call, prompt and reply together; null assumes the protocol default';
