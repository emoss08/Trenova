-- Rolling back discards every usage record and every configured price; calls
-- made afterwards are simply not counted, which is the behaviour before.
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "cost_usd";

--bun:split
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "latency_ms";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "output_cost_per_million";

--bun:split
ALTER TABLE "ai_providers"
    DROP COLUMN IF EXISTS "input_cost_per_million";

--bun:split
DROP INDEX IF EXISTS "idx_ai_usage_records_tenant_time";

--bun:split
DROP TABLE IF EXISTS "ai_usage_records";
