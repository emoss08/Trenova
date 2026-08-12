-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250119183301_location_category.tx.up.sql

CREATE TABLE IF NOT EXISTS "location_categories"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "type" TEXT NOT NULL,
    "facility_type" TEXT,
    "has_secure_parking" INTEGER NOT NULL DEFAULT 0,
    "requires_appointment" INTEGER NOT NULL DEFAULT 0,
    "allows_overnight" INTEGER NOT NULL DEFAULT 0,
    "has_restroom" INTEGER NOT NULL DEFAULT 0,
    "color" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_location_categories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_location_categories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_location_categories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_location_categories_name" ON "location_categories" (lower("name"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_location_categories_business_unit" ON "location_categories" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_location_categories_organization" ON "location_categories" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_location_categories_created_updated" ON "location_categories" ("created_at", "updated_at");
