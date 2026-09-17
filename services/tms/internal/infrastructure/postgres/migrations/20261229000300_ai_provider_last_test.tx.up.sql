-- The outcome of the most recent live probe, so the configuration UI can show
-- whether an endpoint was ever reachable without probing it on every load.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "last_test" jsonb;

--bun:split
COMMENT ON COLUMN "ai_providers"."last_test" IS 'Outcome of the most recent connection test: success, schema adherence, latency and when it ran';
