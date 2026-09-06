-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261019000000_fleet_safety.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_safety_violations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "safety_event_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "basic" TEXT NOT NULL,
    "code" TEXT,
    "description" TEXT NOT NULL,
    "severity_weight" INTEGER NOT NULL DEFAULT 1,
    "out_of_service" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_safety_violations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_safety_violations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_safety_violations_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_worker_safety_violations_weight" CHECK ("severity_weight" >= 1 AND "severity_weight" <= 10)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_event" ON "worker_safety_violations" ("safety_event_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_basic" ON "worker_safety_violations" ("organization_id", "business_unit_id", "basic");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_safety_violations_worker" ON "worker_safety_violations" ("worker_id");

--bun:split

ALTER TABLE "dash_controls" ADD COLUMN "driver_digest_cadence" TEXT NOT NULL DEFAULT 'Immediate';

--bun:split

ALTER TABLE "dash_controls" ADD COLUMN "driver_digest_weekday" INTEGER NOT NULL DEFAULT 1;
