ALTER TABLE "telematics_feed_states"
    DROP CONSTRAINT IF EXISTS "telematics_feed_states_pkey";

ALTER TABLE "telematics_feed_states"
    ADD PRIMARY KEY ("organization_id", "business_unit_id", "feed_type");

ALTER TABLE "telematics_feed_states"
    DROP COLUMN IF EXISTS "provider";

DROP INDEX IF EXISTS idx_telematics_events_provider_event_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_events_event_id
    ON "telematics_events" ("organization_id", "business_unit_id", "event_id");

ALTER TABLE "telematics_events"
    DROP COLUMN IF EXISTS "provider";

ALTER TABLE "worker_hos_states"
    DROP COLUMN IF EXISTS "provider",
    DROP COLUMN IF EXISTS "ruleset_cycle",
    DROP COLUMN IF EXISTS "ruleset_shift",
    DROP COLUMN IF EXISTS "ruleset_restart",
    DROP COLUMN IF EXISTS "ruleset_break",
    DROP COLUMN IF EXISTS "ruleset_jurisdiction";

ALTER TABLE "worker_hos_states"
    RENAME COLUMN "provider_driver_id" TO "samsara_driver_id";

ALTER TABLE "telematics_vehicle_positions"
    DROP COLUMN IF EXISTS "provider";

ALTER TABLE "telematics_vehicle_positions"
    RENAME COLUMN "provider_vehicle_id" TO "samsara_vehicle_id";
