-- The advisor an executor model consulted on Anthropic's side during one call:
-- its model, how many times, its own tokens and what they cost at its list
-- price. The existing token columns stay the executor's alone; cost_usd keeps
-- the whole call, advisor included.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "advisor_model" varchar(200),
    ADD COLUMN IF NOT EXISTS "advisor_calls" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_input_tokens" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_output_tokens" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_cache_read_tokens" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_cache_write_tokens" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_cost_usd" numeric(14, 6);

COMMENT ON COLUMN "ai_usage_records"."advisor_model" IS 'Model of the advisor the executor consulted; null when it consulted none';
COMMENT ON COLUMN "ai_usage_records"."advisor_calls" IS 'Times the executor consulted the advisor during the call';
COMMENT ON COLUMN "ai_usage_records"."advisor_cost_usd" IS 'The advisor''s share of cost_usd, at the advisor model''s list price; null when it consulted none or the advisor has no list price';
