-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250605021341_dedicated_lane.tx.up.sql

CREATE TABLE IF NOT EXISTS "dedicated_lanes"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "service_type_id" TEXT NOT NULL,
    "shipment_type_id" TEXT NOT NULL,
    "trailer_type_id" TEXT,
    "tractor_type_id" TEXT,
    "origin_location_id" TEXT NOT NULL,
    "destination_location_id" TEXT NOT NULL,
    "primary_worker_id" TEXT,
    "secondary_worker_id" TEXT,
    "auto_assign" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dedicated_lanes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dedicated_lanes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_trailer_type" FOREIGN KEY ("trailer_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_tractor_type" FOREIGN KEY ("tractor_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_primary_worker" FOREIGN KEY ("primary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_secondary_worker" FOREIGN KEY ("secondary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_dedicated_lanes_name" ON "dedicated_lanes" (lower("name"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_auto_assign_lookup" ON "dedicated_lanes" ("customer_id", "business_unit_id", "organization_id", "auto_assign")WHERE
    "auto_assign" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_locations" ON "dedicated_lanes" ("origin_location_id", "destination_location_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_equipment" ON "dedicated_lanes" ("tractor_type_id", "trailer_type_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_workers" ON "dedicated_lanes" ("primary_worker_id", "secondary_worker_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_business_unit" ON "dedicated_lanes" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_created_at" ON "dedicated_lanes" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_dedicated_lanes_assignment_lookup ON dedicated_lanes (organization_id, business_unit_id, customer_id, origin_location_id, destination_location_id)WHERE
    auto_assign = TRUE;
