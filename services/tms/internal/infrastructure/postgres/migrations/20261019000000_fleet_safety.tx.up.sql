--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TYPE "csa_basic_enum" AS ENUM(
    'UnsafeDriving',
    'HOSCompliance',
    'DriverFitness',
    'ControlledSubstances',
    'VehicleMaintenance',
    'HazmatCompliance',
    'CrashIndicator'
);

--bun:split
CREATE TYPE "driver_digest_cadence_enum" AS ENUM(
    'Immediate',
    'Daily',
    'Weekly'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_safety_violations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "safety_event_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "basic" csa_basic_enum NOT NULL,
    "code" varchar(20),
    "description" varchar(255) NOT NULL,
    "severity_weight" smallint NOT NULL DEFAULT 1,
    "out_of_service" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_safety_violations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_safety_violations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_safety_violations_weight" CHECK ("severity_weight" >= 1 AND "severity_weight" <= 10)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_event" ON "worker_safety_violations"("safety_event_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_basic" ON "worker_safety_violations"("organization_id", "business_unit_id", "basic");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_worker" ON "worker_safety_violations"("worker_id");

--bun:split
COMMENT ON TABLE worker_safety_violations IS 'One row per violation cited on a safety event. A single roadside inspection routinely produces violations in several BASICs, so the breakdown cannot live on the event as one column without losing most of what an inspection said.';

--bun:split
COMMENT ON COLUMN worker_safety_violations.severity_weight IS 'The FMCSA severity weight, 1 to 10. Out-of-service adds two more when the BASIC is scored, which is why the flag is kept beside it rather than folded in.';

--bun:split
COMMENT ON COLUMN worker_safety_violations.worker_id IS 'Denormalised from the event so a per-driver violation list does not need the join. The event owns the date; nothing here does.';

--bun:split
ALTER TABLE "dash_controls"
    ADD COLUMN IF NOT EXISTS "driver_digest_cadence" driver_digest_cadence_enum NOT NULL DEFAULT 'Immediate';

--bun:split
ALTER TABLE "dash_controls"
    ADD COLUMN IF NOT EXISTS "driver_digest_weekday" smallint NOT NULL DEFAULT 1;

--bun:split
ALTER TABLE "dash_controls"
    ADD CONSTRAINT "chk_dash_controls_digest_weekday" CHECK ("driver_digest_weekday" >= 0 AND "driver_digest_weekday" <= 6);

--bun:split
COMMENT ON COLUMN dash_controls.driver_digest_cadence IS 'How a driver hears about what they owe. Immediate keeps one notice per obligation; Daily and Weekly bundle a driver''s obligations into a single notice. Immediate is the default so no carrier silently loses notices they already rely on.';

--bun:split
COMMENT ON COLUMN dash_controls.driver_digest_weekday IS 'Day the weekly digest goes out, 0 = Sunday. Ignored unless the cadence is Weekly.';
