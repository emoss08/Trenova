-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250114201859_equipment_manufacturer.tx.up.sql

CREATE TABLE IF NOT EXISTS "equipment_manufacturers" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "description" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_equipment_manu" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_equipment_manu_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_equipment_manu_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_equipment_manu_name" ON "equipment_manufacturers" (lower("name"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_equipment_manu_business_unit" ON "equipment_manufacturers" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_equipment_manu_organization" ON "equipment_manufacturers" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_equipment_manu_created_updated" ON "equipment_manufacturers" ("created_at", "updated_at");
