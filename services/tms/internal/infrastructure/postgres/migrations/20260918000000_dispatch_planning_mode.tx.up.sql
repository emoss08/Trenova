CREATE TYPE "dispatch_planning_mode_enum" AS ENUM(
    'Immediate',
    'Horizon'
);

--bun:split
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "planning_mode" "dispatch_planning_mode_enum" NOT NULL DEFAULT 'Immediate',
    ADD COLUMN IF NOT EXISTS "horizon_max_moves_per_driver" SMALLINT NOT NULL DEFAULT 3;

--bun:split
COMMENT ON COLUMN "dispatch_controls"."planning_mode" IS 'How auto-assignment allocates moves; Immediate matches each move to at most one driver against current fleet state, Horizon chains several moves onto one driver and advances that driver position, clock, and trailer after each';

--bun:split
COMMENT ON COLUMN "dispatch_controls"."horizon_max_moves_per_driver" IS 'Maximum moves horizon planning may chain onto a single driver in one run; ignored when planning_mode is Immediate';
