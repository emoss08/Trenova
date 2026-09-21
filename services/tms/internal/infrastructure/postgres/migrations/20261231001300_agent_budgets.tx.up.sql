-- Budgets bound what an agent may spend and do on its own: a cost cap per
-- calendar month, a cap on runs per day, and a cap per tool per day on the
-- writes it executes. Zero or null means no cap. Simulation mode keeps the
-- agent's writes from happening at all: an approved or automatic write is
-- previewed and recorded as what it would have changed, so an agent can be
-- watched at full reach before it is trusted with any of it.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "monthly_budget_usd" numeric(14,6);

--bun:split
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "daily_run_limit" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "tool_daily_limits" jsonb NOT NULL DEFAULT '{}'::jsonb;

--bun:split
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "simulation_mode" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "chk_agent_definitions_budget" CHECK (
        ("monthly_budget_usd" IS NULL OR "monthly_budget_usd" >= 0) AND "daily_run_limit" >= 0
    );

--bun:split
-- What a simulated write would have changed, kept on the proposal in place
-- of an execution.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "simulated_at" bigint;

--bun:split
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "simulation" jsonb;

--bun:split
-- The daily tool cap counts executions of one tool by one agent since the
-- day began; the proposal reaches its agent through its run.
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_executed_tool"
    ON "agent_proposals"("organization_id", "business_unit_id", "tool_name", "executed_at")
    WHERE "executed_at" IS NOT NULL;

COMMENT ON COLUMN "agent_definitions"."monthly_budget_usd" IS 'Cost cap per calendar month across every run of the agent; null for none';

COMMENT ON COLUMN "agent_definitions"."daily_run_limit" IS 'Runs the agent may start per day; 0 for no cap';

COMMENT ON COLUMN "agent_definitions"."tool_daily_limits" IS 'Executions per tool per day, keyed by tool name; absent or 0 for no cap';

COMMENT ON COLUMN "agent_definitions"."simulation_mode" IS 'Writes are previewed and recorded, never made';

COMMENT ON COLUMN "agent_proposals"."simulation" IS 'What the write would have changed, when the agent was in simulation';
