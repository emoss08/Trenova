-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250116013531_shipment_type.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_types"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "description" TEXT,
    "color" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_shipment_types_code" ON "shipment_types" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_types_business_unit" ON "shipment_types" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_types_organization" ON "shipment_types" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_types_created_updated" ON "shipment_types" ("created_at", "updated_at");

--bun:split

ALTER TABLE "shipments" ADD COLUMN "shipment_type_id" TEXT;
