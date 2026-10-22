-- Rolling back turns the advisor and code mode off for every provider.
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_code_mode_writes";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_advisor_kind";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "code_mode_writes";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "code_mode_scripts";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "advisor_refusal";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "advisor_model";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "advisor_enabled";
