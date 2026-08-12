-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260528121000_add_pcmiler_distance_calculation.tx.up.sql

ALTER TABLE "shipment_moves" ADD COLUMN "distance_source" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_provider" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_calculated_at" INTEGER;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_route_signature" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_data_version" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_routing_type" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_units" TEXT;

--bun:split

ALTER TABLE "shipment_moves" ADD COLUMN "distance_metadata" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_distance_route_signature"
    ON "shipment_moves" ("organization_id", "business_unit_id", "distance_route_signature");

--bun:split

CREATE TABLE IF NOT EXISTS "distance_profiles"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "provider" TEXT NOT NULL,
    "data_version" TEXT NOT NULL DEFAULT 'Current',
    "region" TEXT NOT NULL DEFAULT 'NA',
    "routing_type" TEXT NOT NULL DEFAULT 'Practical',
    "distance_units" TEXT NOT NULL DEFAULT 'Miles',
    "location_granularity" TEXT NOT NULL DEFAULT 'PostalCode',
    "profile_name" TEXT,
    "highway_only" INTEGER NOT NULL DEFAULT 0,
    "toll_roads" INTEGER NOT NULL DEFAULT 1,
    "borders_open" INTEGER NOT NULL DEFAULT 1,
    "include_toll_data" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_distance_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_distance_profiles_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_distance_profiles_provider" CHECK ("provider" IN ('PCMiler')),
    CONSTRAINT "ck_distance_profiles_default_active" CHECK ("is_default" = false OR "status" = 'Active')
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_profiles_default"
    ON "distance_profiles" ("organization_id", "business_unit_id")WHERE "is_default" = true;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_distance_profiles_tenant_status"
    ON "distance_profiles" ("organization_id", "business_unit_id", "status", "updated_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "distance_calculation_runs"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "provider" TEXT,
    "source" TEXT NOT NULL,
    "request_summary" TEXT,
    "response_summary" TEXT,
    "status" TEXT NOT NULL,
    "error_code" TEXT,
    "error_message" TEXT,
    "latency_millis" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_distance_calculation_runs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_calculation_runs_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_calculation_runs_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_distance_calculation_runs_shipment"
    ON "distance_calculation_runs" ("organization_id", "business_unit_id", "shipment_id", "created_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "provider_location_resolutions"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "location_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "normalized_address" TEXT,
    "place_id" TEXT,
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_provider_location_resolutions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_provider_location_resolutions_location" FOREIGN KEY ("location_id", "organization_id", "business_unit_id") REFERENCES "locations"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_provider_location_resolutions_location_provider"
    ON "provider_location_resolutions" ("organization_id", "business_unit_id", "location_id", "provider");
