-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250326012748_additional_charge.tx.up.sql

CREATE TABLE IF NOT EXISTS "additional_charges"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "accessorial_charge_id" TEXT NOT NULL,
    "unit" INTEGER NOT NULL,
    "method" TEXT NOT NULL,
    "amount" NUMERIC NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_additional_charges" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id", "accessorial_charge_id"),
    CONSTRAINT "fk_additional_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_accessorial_charge" FOREIGN KEY ("accessorial_charge_id", "business_unit_id", "organization_id") REFERENCES "accessorial_charges"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_shipment" ON "additional_charges" ("shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_accessorial_charge" ON "additional_charges" ("accessorial_charge_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_business_unit" ON "additional_charges" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_organization" ON "additional_charges" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_created_updated" ON "additional_charges" ("created_at", "updated_at");
