-- A unit keeps the advisor's share of its calls, so AI control's Analytics
-- tab can show what consulting the advisor cost and how often a model asked.
-- cost_usd already includes the advisor's cost; advisor_cost_usd is its part.
ALTER TABLE "ai_work_units"
    ADD COLUMN IF NOT EXISTS "advisor_calls" INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_prompt_tokens" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_output_tokens" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "advisor_cost_usd" NUMERIC(14, 6);

-- Units worked out before this carry no advisor figures; the next refresh
-- rebuilds the retention window from the usage rows, which have them.
UPDATE "ai_analytics_refresh_state" SET "refreshed_through" = 0;
