-- One row per attempt to have a model answer: which provider, how long it
-- took, what it consumed and what that cost. Per attempt rather than per turn,
-- because a turn that fell through two providers before a third answered was
-- three calls, three latencies and three bills.
CREATE TABLE IF NOT EXISTS "ai_usage_records" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider_id" varchar(100),
    "provider_kind" varchar(50) NOT NULL,
    "model" varchar(200) NOT NULL,
    "task" varchar(100) NOT NULL,
    "surface" varchar(50) NOT NULL,
    "user_id" varchar(100),
    "agent_definition_id" varchar(100),
    "thread_id" varchar(100),
    "run_id" varchar(100),
    "succeeded" boolean NOT NULL,
    "error_class" varchar(50),
    "streamed" boolean NOT NULL DEFAULT false,
    "latency_ms" bigint NOT NULL,
    "input_tokens" integer NOT NULL DEFAULT 0,
    "output_tokens" integer NOT NULL DEFAULT 0,
    "reasoning_tokens" integer NOT NULL DEFAULT 0,
    "cost_usd" numeric(14, 6),
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_usage_records_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_usage_records_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- The summary asks one question: this tenant, since when. This is its index.
CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_time"
    ON "ai_usage_records"("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
-- What the provider charges, in USD per million tokens, from its price list.
-- Both null means a call's cost is unknown and is reported as such, never as
-- zero.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "input_cost_per_million" numeric(12, 6);

--bun:split
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "output_cost_per_million" numeric(12, 6);

--bun:split
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "latency_ms" bigint;

--bun:split
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "cost_usd" numeric(14, 6);

COMMENT ON TABLE "ai_usage_records" IS 'One row per model call attempt: provider, latency, tokens and cost';

COMMENT ON COLUMN "ai_usage_records"."cost_usd" IS 'Cost at the provider price configured when the call was made; null when the provider carries no price';

COMMENT ON COLUMN "ai_providers"."input_cost_per_million" IS 'USD per million input tokens, from the provider price list; null means unknown';

COMMENT ON COLUMN "ai_providers"."output_cost_per_million" IS 'USD per million output tokens, from the provider price list; null means unknown';

COMMENT ON COLUMN "assistant_messages"."latency_ms" IS 'How long the model took to answer this turn';

COMMENT ON COLUMN "assistant_messages"."cost_usd" IS 'What this turn cost at the provider price; null when unpriced';
