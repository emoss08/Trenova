-- A provider's context window is read off its model id, which is a guess: a
-- self-hosted model can be served with a shorter or longer context than its
-- family's, and an id nothing recognizes gets a default. An operator who knows
-- the window can now say so. Null keeps reading it off the id.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "context_window" integer;

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_context_window" CHECK ("context_window" IS NULL OR "context_window" BETWEEN 4096 AND 10000000);

--bun:split
COMMENT ON COLUMN "ai_providers"."context_window" IS 'Context window of the model in tokens, as the operator configured it; null reads it off the model id';
