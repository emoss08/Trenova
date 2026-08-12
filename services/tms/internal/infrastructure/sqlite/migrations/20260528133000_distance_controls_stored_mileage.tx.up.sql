-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260528133000_distance_controls_stored_mileage.tx.up.sql

CREATE TABLE IF NOT EXISTS "distance_controls"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "store_mileage" INTEGER NOT NULL DEFAULT 1,
    "stored_distance_units" TEXT NOT NULL DEFAULT 'Miles',
    "postal_code_fallback_to_city" INTEGER NOT NULL DEFAULT 1,
    "auto_create_stored_mileage" INTEGER NOT NULL DEFAULT 1,
    "loaded_move_distance_profile_id" TEXT NOT NULL,
    "empty_move_distance_profile_id" TEXT NOT NULL,
    "pay_distance_profile_id" TEXT NOT NULL,
    "billing_distance_profile_id" TEXT NOT NULL,
    "fuel_distance_profile_id" TEXT NOT NULL,
    "eta_out_of_route_distance_profile_id" TEXT NOT NULL,
    "distance_calculator_shortest_distance_profile_id" TEXT NOT NULL,
    "distance_calculator_practical_distance_profile_id" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_distance_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_distance_controls_stored_units" CHECK ("stored_distance_units" IN ('Miles', 'Kilometers'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_controls_tenant"
    ON "distance_controls" ("organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "stored_mileages"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "origin_key" TEXT NOT NULL,
    "destination_key" TEXT NOT NULL,
    "intermediate_keys" TEXT,
    "route_signature" TEXT NOT NULL,
    "route_hash" TEXT NOT NULL,
    "distance" REAL NOT NULL,
    "distance_units" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "routing_type" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "location_granularity" TEXT NOT NULL,
    "data_version" TEXT NOT NULL,
    "distance_profile_id" TEXT NOT NULL,
    "distance_profile_name" TEXT,
    "hazmat" INTEGER NOT NULL DEFAULT 0,
    "hazmat_types" TEXT,
    "hazmat_signature" TEXT NOT NULL DEFAULT 'none',
    "provider_metadata" TEXT,
    "hit_count" INTEGER NOT NULL DEFAULT 0,
    "last_used_at" INTEGER,
    "last_calculated_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_stored_mileages" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stored_mileages_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stored_mileages_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_stored_mileages_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_stored_mileages_distance" CHECK ("distance" >= 0),
    CONSTRAINT "ck_stored_mileages_units" CHECK ("distance_units" IN ('Miles', 'Kilometers'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_stored_mileages_active_policy"
    ON "stored_mileages" (
        "organization_id",
        "business_unit_id",
        "route_hash",
        "distance_units",
        "routing_type",
        "method",
        "distance_profile_id",
        "hazmat_signature"
    )WHERE "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_stored_mileages_tenant_status"
    ON "stored_mileages" ("organization_id", "business_unit_id", "status", "updated_at" DESC);
