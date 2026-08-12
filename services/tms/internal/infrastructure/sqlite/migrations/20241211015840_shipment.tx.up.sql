-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015840_shipment.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipments"(
    "id" TEXT NOT NULL,
    "pro_number" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'New',
    "bol" TEXT,
    "actual_ship_date" INTEGER,
    "actual_delivery_date" INTEGER,
    "temperature_min" INTEGER,
    "temperature_max" INTEGER,
    "rating_unit" INTEGER NOT NULL DEFAULT 1 CHECK ("rating_unit" > 0),
    "freight_charge_amount" NUMERIC NOT NULL DEFAULT 0 CHECK ("freight_charge_amount" >= 0),
    "other_charge_amount" NUMERIC NOT NULL DEFAULT 0 CHECK ("other_charge_amount" >= 0),
    "total_charge_amount" NUMERIC NOT NULL DEFAULT 0 CHECK ("total_charge_amount" >= 0),
    "pieces" INTEGER,
    "weight" INTEGER,
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "cancel_reason" TEXT,
    "entered_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_entered_by" FOREIGN KEY ("entered_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_shipments_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_bol" ON "shipments" ("bol");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_created_at" ON "shipments" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_status" ON "shipments" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_entered_by" ON "shipments" ("entered_by_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_business_unit" ON "shipments" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipments_active ON shipments (organization_id, created_at DESC)WHERE
    status NOT IN ('Completed', 'Invoiced', 'Canceled');

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipments_in_transit ON shipments (organization_id, business_unit_id)WHERE
    status = 'InTransit';

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipments_bu_org_status_created_at ON shipments (business_unit_id, organization_id, status, created_at DESC);

--bun:split

ALTER TABLE "shipments" ADD COLUMN "owner_id" TEXT;
