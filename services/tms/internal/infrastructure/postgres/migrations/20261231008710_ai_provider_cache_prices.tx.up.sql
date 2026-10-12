-- A provider's cache prices. Most of a Desk turn's prompt is read from the
-- provider's cache: priced as fresh input it overstated an OpenAI call, and
-- left out (Anthropic counts cached tokens apart from input) it understated an
-- Anthropic one. Null takes the protocol's usual multiple of the input price.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "cache_read_cost_per_million" numeric(12, 6);

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "cache_write_cost_per_million" numeric(12, 6);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_cache_read_cost_non_negative"
    CHECK ("cache_read_cost_per_million" IS NULL OR "cache_read_cost_per_million" >= 0);

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_cache_write_cost_non_negative"
    CHECK ("cache_write_cost_per_million" IS NULL OR "cache_write_cost_per_million" >= 0);

--bun:split
COMMENT ON COLUMN "ai_providers"."cache_read_cost_per_million" IS 'USD per million prompt tokens read from the provider cache; null takes the protocol default share of the input price';

--bun:split
COMMENT ON COLUMN "ai_providers"."cache_write_cost_per_million" IS 'USD per million prompt tokens written to the provider cache; null takes the protocol default multiple of the input price';
