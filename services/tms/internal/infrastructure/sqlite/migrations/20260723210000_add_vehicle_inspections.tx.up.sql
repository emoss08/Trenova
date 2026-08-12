-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260723210000_add_vehicle_inspections.tx.up.sql

ALTER TABLE "trailers" ADD COLUMN "external_id" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_trailers_external_id_tenant
    ON "trailers" ("organization_id", "business_unit_id", "external_id")WHERE "external_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "vehicle_inspections"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL DEFAULT 'Samsara',
    "provider_dvir_id" TEXT NOT NULL,
    "tractor_id" TEXT,
    "worker_id" TEXT,
    "inspection_type" TEXT NOT NULL,
    "safety_status" TEXT NOT NULL,
    "started_at" INTEGER NOT NULL,
    "ended_at" INTEGER NOT NULL,
    "odometer_meters" INTEGER,
    "location" TEXT,
    "signed" INTEGER NOT NULL DEFAULT 0,
    "defect_count" INTEGER NOT NULL DEFAULT 0,
    "unresolved_defect_count" INTEGER NOT NULL DEFAULT 0,
    "defects" TEXT,
    "created_at" INTEGER NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_vehicle_inspections_provider_dvir_id
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "provider", "provider_dvir_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_started_at
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "started_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_tractor
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "tractor_id")WHERE "tractor_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_worker
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "worker_id")WHERE "worker_id" IS NOT NULL;
