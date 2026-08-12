-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260723200000_telematics_multiprovider.tx.up.sql

ALTER TABLE "telematics_vehicle_positions" RENAME COLUMN "samsara_vehicle_id" TO "provider_vehicle_id";

--bun:split

ALTER TABLE "telematics_vehicle_positions" ADD COLUMN "provider" TEXT NOT NULL DEFAULT 'Samsara';

--bun:split

ALTER TABLE "worker_hos_states" RENAME COLUMN "samsara_driver_id" TO "provider_driver_id";

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "provider" TEXT NOT NULL DEFAULT 'Samsara';

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "ruleset_cycle" TEXT;

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "ruleset_shift" TEXT;

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "ruleset_restart" TEXT;

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "ruleset_break" TEXT;

--bun:split

ALTER TABLE "worker_hos_states" ADD COLUMN "ruleset_jurisdiction" TEXT;

--bun:split

ALTER TABLE "telematics_events" ADD COLUMN "provider" TEXT NOT NULL DEFAULT 'Samsara';

--bun:split

DROP INDEX IF EXISTS idx_telematics_events_event_id;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_events_provider_event_id
    ON "telematics_events" ("organization_id", "business_unit_id", "provider", "event_id");

--bun:split

ALTER TABLE "telematics_feed_states" ADD COLUMN "provider" TEXT NOT NULL DEFAULT 'Samsara';
