ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "horizon_max_moves_per_driver",
    DROP COLUMN IF EXISTS "planning_mode";

--bun:split
DROP TYPE IF EXISTS "dispatch_planning_mode_enum";
