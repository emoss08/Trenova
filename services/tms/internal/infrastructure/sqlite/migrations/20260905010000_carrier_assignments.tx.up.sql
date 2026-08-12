-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260905010000_carrier_assignments.tx.up.sql

ALTER TABLE "shipment_moves" ADD COLUMN "coverage_type" TEXT NOT NULL DEFAULT 'driver';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves" ("coverage_type", "organization_id")WHERE
    "coverage_type" <> 'driver';

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_assignments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "carrier_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "rate_method" TEXT NOT NULL DEFAULT 'Flat',
    "base_rate" REAL NOT NULL DEFAULT 0,
    "base_amount" REAL NOT NULL DEFAULT 0,
    "fuel_surcharge" REAL NOT NULL DEFAULT 0,
    "accessorial_total" REAL NOT NULL DEFAULT 0,
    "total_cost" REAL NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "pro_number" TEXT,
    "external_driver_name" TEXT,
    "external_driver_phone" TEXT,
    "external_tractor_number" TEXT,
    "external_trailer_number" TEXT,
    "assigned_by_id" TEXT,
    "confirmed_at" INTEGER,
    "canceled_at" INTEGER,
    "cancellation_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_assignments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_assignments_amounts" CHECK ("base_rate" >= 0 AND "fuel_surcharge" >= 0 AND "accessorial_total" >= 0 AND "total_cost" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_carrier_assignments_active_move" ON "carrier_assignments" ("shipment_move_id", "organization_id", "business_unit_id")WHERE
    "status" <> 'Canceled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_assignments_carrier" ON "carrier_assignments" ("carrier_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_assignments_status" ON "carrier_assignments" ("status", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_assignments_pro_number" ON "carrier_assignments" ("pro_number", "organization_id")WHERE
    "pro_number" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "carrier_assignment_accessorials"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "carrier_assignment_id" TEXT NOT NULL,
    "accessorial_charge_id" TEXT,
    "description" TEXT NOT NULL,
    "amount" REAL NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_carrier_assignment_accessorials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_assignment_accessorials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignment_accessorials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignment_accessorials_assignment" FOREIGN KEY ("carrier_assignment_id", "organization_id", "business_unit_id") REFERENCES "carrier_assignments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_assignment_accessorials_amount" CHECK ("amount" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carrier_assignment_accessorials_assignment" ON "carrier_assignment_accessorials" ("carrier_assignment_id", "organization_id");
