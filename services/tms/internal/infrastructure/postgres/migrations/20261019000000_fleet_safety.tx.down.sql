ALTER TABLE "dash_controls"
    DROP CONSTRAINT IF EXISTS "chk_dash_controls_digest_weekday";
--bun:split
ALTER TABLE "dash_controls"
    DROP COLUMN IF EXISTS "driver_digest_weekday";
--bun:split
ALTER TABLE "dash_controls"
    DROP COLUMN IF EXISTS "driver_digest_cadence";
--bun:split
DROP TABLE IF EXISTS "worker_safety_violations";
--bun:split
DROP TYPE IF EXISTS "driver_digest_cadence_enum";
--bun:split
DROP TYPE IF EXISTS "csa_basic_enum";
