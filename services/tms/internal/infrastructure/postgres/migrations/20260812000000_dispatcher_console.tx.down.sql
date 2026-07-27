--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_max_deadhead";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_planning_horizon";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_auto_assign_confidence";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "auto_assign_planning_horizon_hours",
    DROP COLUMN IF EXISTS "auto_assign_max_deadhead_miles",
    DROP COLUMN IF EXISTS "auto_assign_confidence_threshold",
    DROP COLUMN IF EXISTS "scoring_weights";
