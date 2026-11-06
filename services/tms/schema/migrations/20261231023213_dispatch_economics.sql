-- Dispatch speaks money: each organization's costs (empty miles, late minutes,
-- driver hours, home time, fairness), its soft preferences in cents and how it
-- prices leaving a move uncovered replace the relative scoring weights and the
-- strategy presets they hung off. Learning the organization's dispatch style
-- and planning for where drivers end up are switches, off until turned on.
ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "scoring_weights",
    DROP COLUMN IF EXISTS "auto_assignment_strategy",
    ADD COLUMN IF NOT EXISTS "economics" jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "learn_dispatch_style" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "plan_for_end_market" boolean NOT NULL DEFAULT false;

DROP TYPE IF EXISTS "auto_assignment_strategy_enum";

COMMENT ON COLUMN "dispatch_controls"."economics" IS 'Costs and prices the dispatch optimizer plans with, in cents: empty mile, late minute, driver hour, home time, fairness, soft preferences and the price of leaving a move uncovered';
COMMENT ON COLUMN "dispatch_controls"."learn_dispatch_style" IS 'Learn how much this organization''s dispatchers weigh each preference from their approvals, declines and overrides';
COMMENT ON COLUMN "dispatch_controls"."plan_for_end_market" IS 'Price where a driver ends up from the planner''s own history of driver values by market and hour of week';
