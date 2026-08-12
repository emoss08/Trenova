-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250122212814_location.tx.up.sql

CREATE TABLE IF NOT EXISTS "locations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "location_category_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT,
    "description" TEXT,
    "address_line_1" TEXT,
    "address_line_2" TEXT,
    "city" TEXT,
    "postal_code" TEXT,
    "longitude" TEXT,
    "latitude" TEXT,
    "place_id" TEXT,
    "is_geocoded" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_locations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_locations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_locations_location_category" FOREIGN KEY ("location_category_id", "business_unit_id", "organization_id") REFERENCES "location_categories"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_locations_code" ON "locations" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_locations_name" ON "locations" ("name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_locations_business_unit" ON "locations" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_locations_organization" ON "locations" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_locations_created_updated" ON "locations" ("created_at", "updated_at");

--bun:split

ALTER TABLE "stops" ADD COLUMN "location_id" TEXT NOT NULL;
