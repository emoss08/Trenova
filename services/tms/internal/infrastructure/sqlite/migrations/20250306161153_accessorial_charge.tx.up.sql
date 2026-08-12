-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250306161153_accessorial_charge.tx.up.sql

CREATE TABLE IF NOT EXISTS "accessorial_charges"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "rate_unit" TEXT,
    "method" TEXT NOT NULL,
    "amount" NUMERIC NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accessorial_charges" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_accessorial_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accessorial_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_status" ON "accessorial_charges" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_business_unit" ON "accessorial_charges" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_accessorial_charges_active ON accessorial_charges (created_at DESC)WHERE
    status != 'Inactive';

--bun:split

ALTER TABLE "shipment_controls" ADD COLUMN "detention_charge_id" TEXT;
