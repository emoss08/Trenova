-- Rolling back reads every Claude model's thinking style from its id again,
-- which is the behaviour before this migration.
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_thinking_style_kind";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_thinking_style";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "thinking_style";
