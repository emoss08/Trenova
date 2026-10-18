ALTER TABLE "ai_work_units"
    DROP COLUMN IF EXISTS "cache_write_tokens",
    DROP COLUMN IF EXISTS "cache_read_tokens";
