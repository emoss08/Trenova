ALTER TABLE "tractors"
    ADD COLUMN IF NOT EXISTS "external_id" text;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tractors_external_id_tenant
    ON "tractors" ("organization_id", "business_unit_id", "external_id")
    WHERE "external_id" IS NOT NULL;

CREATE TABLE IF NOT EXISTS "telematics_vehicle_positions"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "tractor_id" character varying(100) NOT NULL,
    "samsara_vehicle_id" text NOT NULL,
    "latitude" double precision NOT NULL,
    "longitude" double precision NOT NULL,
    "heading_degrees" double precision NOT NULL DEFAULT 0,
    "speed_mph" double precision NOT NULL DEFAULT 0,
    "engine_state" character varying(16),
    "fuel_percent" double precision,
    "odometer_meters" bigint,
    "formatted_location" text,
    "recorded_at" bigint NOT NULL,
    "received_at" bigint NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "tractor_id")
);

CREATE INDEX IF NOT EXISTS idx_telematics_vehicle_positions_recorded_at
    ON "telematics_vehicle_positions" ("organization_id", "business_unit_id", "recorded_at" DESC);

CREATE TABLE IF NOT EXISTS "worker_hos_states"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "worker_id" character varying(100) NOT NULL,
    "samsara_driver_id" text NOT NULL,
    "duty_status" character varying(32),
    "drive_remaining_ms" bigint NOT NULL DEFAULT 0,
    "shift_remaining_ms" bigint NOT NULL DEFAULT 0,
    "cycle_remaining_ms" bigint NOT NULL DEFAULT 0,
    "cycle_tomorrow_ms" bigint NOT NULL DEFAULT 0,
    "break_remaining_ms" bigint NOT NULL DEFAULT 0,
    "cycle_started_at" bigint,
    "shift_driving_violation_ms" bigint NOT NULL DEFAULT 0,
    "cycle_violation_ms" bigint NOT NULL DEFAULT 0,
    "current_vehicle_id" text,
    "current_tractor_id" character varying(100),
    "recorded_at" bigint NOT NULL,
    "received_at" bigint NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id")
);

CREATE INDEX IF NOT EXISTS idx_worker_hos_states_drive_remaining
    ON "worker_hos_states" ("organization_id", "business_unit_id", "drive_remaining_ms" ASC);

CREATE TABLE IF NOT EXISTS "worker_hos_violations"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "worker_id" character varying(100) NOT NULL,
    "violation_type" character varying(64) NOT NULL,
    "violation_start_at" bigint NOT NULL,
    "description" text,
    "duration_ms" bigint NOT NULL DEFAULT 0,
    "day_start_at" bigint,
    "day_end_at" bigint,
    "detected_at" bigint NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "violation_type", "violation_start_at")
);

CREATE INDEX IF NOT EXISTS idx_worker_hos_violations_start_at
    ON "worker_hos_violations" ("organization_id", "business_unit_id", "violation_start_at" DESC);

CREATE TABLE IF NOT EXISTS "telematics_events"(
    "id" character varying(100) NOT NULL,
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "event_id" text NOT NULL,
    "event_type" character varying(64) NOT NULL,
    "occurred_at" bigint NOT NULL,
    "tractor_id" character varying(100),
    "worker_id" character varying(100),
    "location_id" character varying(100),
    "address_name" text,
    "payload" jsonb,
    "created_at" bigint NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_events_event_id
    ON "telematics_events" ("organization_id", "business_unit_id", "event_id");

CREATE INDEX IF NOT EXISTS idx_telematics_events_occurred_at
    ON "telematics_events" ("organization_id", "business_unit_id", "occurred_at" DESC);

CREATE TABLE IF NOT EXISTS "telematics_feed_states"(
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "feed_type" character varying(32) NOT NULL,
    "cursor" text,
    "last_polled_at" bigint,
    "last_success_at" bigint,
    "failure_count" integer NOT NULL DEFAULT 0,
    "last_error" text,
    PRIMARY KEY ("organization_id", "business_unit_id", "feed_type")
);
