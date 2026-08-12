-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250116043759_hazardous_material.tx.up.sql

CREATE TABLE IF NOT EXISTS "hazardous_materials"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "class" TEXT NOT NULL,
    "un_number" TEXT,
    "packing_group" TEXT NOT NULL,
    "subsidiary_hazard_class" TEXT,
    "erg_guide_number" TEXT,
    "label_codes" TEXT,
    "special_provisions" TEXT,
    "proper_shipping_name" TEXT,
    "handling_instructions" TEXT,
    "emergency_contact" TEXT,
    "emergency_contact_phone_number" TEXT,
    "quantity_threshold" TEXT,
    "placard_required" INTEGER NOT NULL DEFAULT 0,
    "is_reportable_quantity" INTEGER NOT NULL DEFAULT 0,
    "marine_pollutant" INTEGER NOT NULL DEFAULT 0,
    "inhalation_hazard" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_hazardous_materials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_hazardous_materials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazardous_materials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_hazardous_materials_code" ON "hazardous_materials" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazardous_materials_business_unit" ON "hazardous_materials" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazardous_materials_organization" ON "hazardous_materials" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hazardous_materials_created_updated" ON "hazardous_materials" ("created_at", "updated_at");
