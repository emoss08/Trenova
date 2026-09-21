-- A failed attempt used to record only the class of failure. The provider's
-- own message is what says why: a provider that is down and a provider that
-- refuses every request both showed as "provider_error", and the difference
-- decides whether an administrator waits or fixes the configuration.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "error_message" varchar(500);

--bun:split
-- The overview lists the newest failures; this is the index it reads.
CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_failures"
    ON "ai_usage_records"("organization_id", "business_unit_id", "created_at" DESC)
    WHERE "succeeded" = false;

COMMENT ON COLUMN "ai_usage_records"."error_message" IS 'The provider''s own message for a failed attempt, cut to 500 characters; empty on success';
