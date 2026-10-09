ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "fk_ai_providers_api_key_added_by",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_previous_api_key",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_api_key_last_four",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_on_cap",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_monthly_cap_usd",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_max_concurrent",
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_timeout_seconds";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "rotation_expires_at",
    DROP COLUMN IF EXISTS "previous_api_key",
    DROP COLUMN IF EXISTS "api_key_last_used_at",
    DROP COLUMN IF EXISTS "api_key_added_by_id",
    DROP COLUMN IF EXISTS "api_key_added_at",
    DROP COLUMN IF EXISTS "api_key_last_four",
    DROP COLUMN IF EXISTS "api_key_prefix",
    DROP COLUMN IF EXISTS "on_cap",
    DROP COLUMN IF EXISTS "monthly_cap_usd",
    DROP COLUMN IF EXISTS "max_concurrent",
    DROP COLUMN IF EXISTS "timeout_seconds";
