ALTER TABLE "ai_work_units"
    ADD COLUMN IF NOT EXISTS "cache_read_tokens" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "cache_write_tokens" BIGINT NOT NULL DEFAULT 0;

UPDATE "ai_analytics_refresh_state" SET "refreshed_through" = 0;
