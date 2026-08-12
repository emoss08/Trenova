-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250118235053_shipment_commodity.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_commodities"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "commodity_id" TEXT NOT NULL,
    "weight" INTEGER NOT NULL,
    "pieces" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_commodities" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id", "commodity_id"),
    CONSTRAINT "fk_shipment_commodities_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_commodity" FOREIGN KEY ("commodity_id", "business_unit_id", "organization_id") REFERENCES "commodities"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_commodities_shipment" ON "shipment_commodities" ("shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_commodities_commodity" ON "shipment_commodities" ("commodity_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_commodities_business_unit" ON "shipment_commodities" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_commodities_organization" ON "shipment_commodities" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_commodities_created_updated" ON "shipment_commodities" ("created_at", "updated_at");
