--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

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
