ALTER TABLE "trailers"
    ADD COLUMN IF NOT EXISTS "external_id" text;

CREATE UNIQUE INDEX IF NOT EXISTS idx_trailers_external_id_tenant
    ON "trailers" ("organization_id", "business_unit_id", "external_id")
    WHERE "external_id" IS NOT NULL;

CREATE TABLE IF NOT EXISTS "vehicle_inspections"(
    "id" character varying(100) NOT NULL,
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "provider" character varying(32) NOT NULL DEFAULT 'Samsara',
    "provider_dvir_id" text NOT NULL,
    "tractor_id" character varying(100),
    "worker_id" character varying(100),
    "inspection_type" character varying(32) NOT NULL,
    "safety_status" character varying(16) NOT NULL,
    "started_at" bigint NOT NULL,
    "ended_at" bigint NOT NULL,
    "odometer_meters" bigint,
    "location" text,
    "signed" boolean NOT NULL DEFAULT FALSE,
    "defect_count" integer NOT NULL DEFAULT 0,
    "unresolved_defect_count" integer NOT NULL DEFAULT 0,
    "defects" jsonb,
    "created_at" bigint NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vehicle_inspections_provider_dvir_id
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "provider", "provider_dvir_id");

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_started_at
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "started_at" DESC);

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_tractor
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "tractor_id")
    WHERE "tractor_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_vehicle_inspections_worker
    ON "vehicle_inspections" ("organization_id", "business_unit_id", "worker_id")
    WHERE "worker_id" IS NOT NULL;
