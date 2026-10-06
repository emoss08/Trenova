-- How long an attempt took to write the first piece of its reply, text or
-- thinking. An attempt's latency runs to the end of the reply, so a fast
-- model writing a long answer read as slow; this is the wait a person sees.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "first_token_ms" bigint;

COMMENT ON COLUMN "ai_usage_records"."first_token_ms" IS 'Milliseconds from the attempt starting to the first text or thinking it streamed; null when it streamed none';
