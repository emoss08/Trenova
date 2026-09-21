-- Rolling back discards every saved reasoning trace and turns reasoning off
-- for every provider, which is the behaviour before this migration.
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "reasoning";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_reasoning_effort";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "reasoning_effort";
