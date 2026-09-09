-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261101000100_shipment_move_jurisdiction_miles.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_move_jurisdiction_miles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "country_code" TEXT NOT NULL,
    "jurisdiction_code" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL DEFAULT 0,
    "distance" REAL NOT NULL DEFAULT 0,
    "distance_units" TEXT NOT NULL DEFAULT 'Miles',
    "toll_distance" REAL,
    "ferry_distance" REAL,
    "loaded" INTEGER NOT NULL DEFAULT 1,
    "source" TEXT NOT NULL DEFAULT 'RouteCalculation',
    "provider" TEXT,
    "data_version" TEXT,
    "distance_profile_id" TEXT,
    "calculated_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_move_jurisdiction_miles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_distance" CHECK ("distance" >= 0),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_units" CHECK ("distance_units" IN ('Miles', 'Kilometers')),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_source" CHECK ("source" IN ('RouteCalculation', 'Manual')),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_country" CHECK ("country_code" = upper("country_code") AND length("country_code") = 2)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipment_move_jurisdiction_miles_move_jurisdiction" ON "shipment_move_jurisdiction_miles" ("organization_id", "business_unit_id", "shipment_move_id", "country_code", "jurisdiction_code");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_move_jurisdiction_miles_jurisdiction" ON "shipment_move_jurisdiction_miles" ("organization_id", "business_unit_id", "country_code", "jurisdiction_code", "calculated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_move_jurisdiction_miles_shipment" ON "shipment_move_jurisdiction_miles" ("organization_id", "business_unit_id", "shipment_id");

--bun:split

ALTER TABLE "stored_mileages" ADD COLUMN "jurisdiction_distances" TEXT;

--bun:split

ALTER TABLE "distance_controls" ADD COLUMN "capture_jurisdiction_miles" INTEGER NOT NULL DEFAULT 0;
