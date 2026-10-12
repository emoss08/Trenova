-- Rolling back calls every provider at its base URL again. A provider set up on a
-- platform keeps its row but loses its platform settings.
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform_settings";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform_kind";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "platform_settings",
    DROP COLUMN IF EXISTS "platform";
