-- An attempt answered through a provider's batch API (Anthropic's Message
-- Batches), priced at the batch discount. cost_usd on such a row is already
-- discounted; the flag says why it is lower than the same tokens bought one
-- call at a time.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "batched" boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN "ai_usage_records"."batched" IS 'Whether the attempt ran through the provider batch API; cost_usd is then at the batch discount';
