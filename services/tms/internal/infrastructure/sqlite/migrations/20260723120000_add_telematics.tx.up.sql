-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260723120000_add_telematics.tx.up.sql

ALTER TABLE "tractors" ADD COLUMN "external_id" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_tractors_external_id_tenant
    ON "tractors" ("organization_id", "business_unit_id", "external_id")WHERE "external_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_vehicle_positions"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "tractor_id" TEXT NOT NULL,
    "samsara_vehicle_id" TEXT NOT NULL,
    "latitude" REAL NOT NULL,
    "longitude" REAL NOT NULL,
    "heading_degrees" REAL NOT NULL DEFAULT 0,
    "speed_mph" REAL NOT NULL DEFAULT 0,
    "engine_state" TEXT,
    "fuel_percent" REAL,
    "odometer_meters" INTEGER,
    "formatted_location" TEXT,
    "recorded_at" INTEGER NOT NULL,
    "received_at" INTEGER NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "tractor_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_telematics_vehicle_positions_recorded_at
    ON "telematics_vehicle_positions" ("organization_id", "business_unit_id", "recorded_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_hos_states"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "samsara_driver_id" TEXT NOT NULL,
    "duty_status" TEXT,
    "drive_remaining_ms" INTEGER NOT NULL DEFAULT 0,
    "shift_remaining_ms" INTEGER NOT NULL DEFAULT 0,
    "cycle_remaining_ms" INTEGER NOT NULL DEFAULT 0,
    "cycle_tomorrow_ms" INTEGER NOT NULL DEFAULT 0,
    "break_remaining_ms" INTEGER NOT NULL DEFAULT 0,
    "cycle_started_at" INTEGER,
    "shift_driving_violation_ms" INTEGER NOT NULL DEFAULT 0,
    "cycle_violation_ms" INTEGER NOT NULL DEFAULT 0,
    "current_vehicle_id" TEXT,
    "current_tractor_id" TEXT,
    "recorded_at" INTEGER NOT NULL,
    "received_at" INTEGER NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_hos_states_drive_remaining
    ON "worker_hos_states" ("organization_id", "business_unit_id", "drive_remaining_ms" ASC);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_hos_violations"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "violation_type" TEXT NOT NULL,
    "violation_start_at" INTEGER NOT NULL,
    "description" TEXT,
    "duration_ms" INTEGER NOT NULL DEFAULT 0,
    "day_start_at" INTEGER,
    "day_end_at" INTEGER,
    "detected_at" INTEGER NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "violation_type", "violation_start_at")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_hos_violations_start_at
    ON "worker_hos_violations" ("organization_id", "business_unit_id", "violation_start_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_events"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "event_type" TEXT NOT NULL,
    "occurred_at" INTEGER NOT NULL,
    "tractor_id" TEXT,
    "worker_id" TEXT,
    "location_id" TEXT,
    "address_name" TEXT,
    "payload" TEXT,
    "created_at" INTEGER NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_events_event_id
    ON "telematics_events" ("organization_id", "business_unit_id", "event_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_telematics_events_occurred_at
    ON "telematics_events" ("organization_id", "business_unit_id", "occurred_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_feed_states"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "feed_type" TEXT NOT NULL,
    "cursor" TEXT,
    "last_polled_at" INTEGER,
    "last_success_at" INTEGER,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    PRIMARY KEY ("organization_id", "business_unit_id", "feed_type")
);
