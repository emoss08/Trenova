ALTER TABLE "telematics_vehicle_positions"
    RENAME COLUMN "samsara_vehicle_id" TO "provider_vehicle_id";

ALTER TABLE "telematics_vehicle_positions"
    ADD COLUMN IF NOT EXISTS "provider" character varying(32) NOT NULL DEFAULT 'Samsara';

ALTER TABLE "worker_hos_states"
    RENAME COLUMN "samsara_driver_id" TO "provider_driver_id";

ALTER TABLE "worker_hos_states"
    ADD COLUMN IF NOT EXISTS "provider" character varying(32) NOT NULL DEFAULT 'Samsara',
    ADD COLUMN IF NOT EXISTS "ruleset_cycle" character varying(64),
    ADD COLUMN IF NOT EXISTS "ruleset_shift" character varying(64),
    ADD COLUMN IF NOT EXISTS "ruleset_restart" character varying(64),
    ADD COLUMN IF NOT EXISTS "ruleset_break" character varying(64),
    ADD COLUMN IF NOT EXISTS "ruleset_jurisdiction" character varying(16);

ALTER TABLE "telematics_events"
    ADD COLUMN IF NOT EXISTS "provider" character varying(32) NOT NULL DEFAULT 'Samsara';

DROP INDEX IF EXISTS idx_telematics_events_event_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_events_provider_event_id
    ON "telematics_events" ("organization_id", "business_unit_id", "provider", "event_id");

ALTER TABLE "telematics_feed_states"
    ADD COLUMN IF NOT EXISTS "provider" character varying(32) NOT NULL DEFAULT 'Samsara';

ALTER TABLE "telematics_feed_states"
    DROP CONSTRAINT IF EXISTS "telematics_feed_states_pkey";

ALTER TABLE "telematics_feed_states"
    ADD PRIMARY KEY ("organization_id", "business_unit_id", "provider", "feed_type");
