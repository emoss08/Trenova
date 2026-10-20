ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "advisor_cost_usd",
    DROP COLUMN IF EXISTS "advisor_cache_write_tokens",
    DROP COLUMN IF EXISTS "advisor_cache_read_tokens",
    DROP COLUMN IF EXISTS "advisor_output_tokens",
    DROP COLUMN IF EXISTS "advisor_input_tokens",
    DROP COLUMN IF EXISTS "advisor_calls",
    DROP COLUMN IF EXISTS "advisor_model";
