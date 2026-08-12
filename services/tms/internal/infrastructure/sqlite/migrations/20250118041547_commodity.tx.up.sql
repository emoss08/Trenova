-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250118041547_commodity.tx.up.sql

CREATE TABLE IF NOT EXISTS "commodities"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "hazardous_material_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "min_temperature" INTEGER,
    "max_temperature" INTEGER,
    "max_quantity_per_shipment" REAL,
    "weight_per_unit" REAL,
    "linear_feet_per_unit" REAL,
    "freight_class" TEXT,
    "loading_instructions" TEXT,
    "stackable" INTEGER NOT NULL DEFAULT 0,
    "fragile" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_commodities" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_commodities_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_commodities_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_commodities_hazardous_material" FOREIGN KEY ("hazardous_material_id", "business_unit_id", "organization_id") REFERENCES "hazardous_materials"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_commodities_name" ON "commodities" (lower("name"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_commodities_business_unit" ON "commodities" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_commodities_organization" ON "commodities" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_commodities_created_updated" ON "commodities" ("created_at", "updated_at");
