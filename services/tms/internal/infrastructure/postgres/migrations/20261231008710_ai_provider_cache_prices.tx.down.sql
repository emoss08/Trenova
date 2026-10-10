-- Rolling back prices every cached token at the protocol default again.
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_cache_write_cost_non_negative";

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_cache_read_cost_non_negative";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "cache_write_cost_per_million";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "cache_read_cost_per_million";
