--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "scoring_weights" JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "auto_assign_confidence_threshold" NUMERIC(5, 4) NOT NULL DEFAULT 0.8500,
    ADD COLUMN IF NOT EXISTS "auto_assign_max_deadhead_miles" INTEGER,
    ADD COLUMN IF NOT EXISTS "auto_assign_planning_horizon_hours" SMALLINT NOT NULL DEFAULT 48;

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_auto_assign_confidence";

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_auto_assign_confidence" CHECK (
        "auto_assign_confidence_threshold" >= 0
        AND "auto_assign_confidence_threshold" <= 1
    );

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_planning_horizon";

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_planning_horizon" CHECK (
        "auto_assign_planning_horizon_hours" > 0
        AND "auto_assign_planning_horizon_hours" <= 336
    );

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_max_deadhead";

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_max_deadhead" CHECK (
        "auto_assign_max_deadhead_miles" IS NULL
        OR "auto_assign_max_deadhead_miles" > 0
    );

--bun:split
-- The dispatch agent is opt-in per organization, separately from the billing agent, so
-- turning one on never turns the other on.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "dispatch_agent_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "dispatch_autonomy_tier" agent_autonomy_tier_enum NOT NULL DEFAULT 'Propose';
