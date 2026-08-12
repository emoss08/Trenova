-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250125144620_tractor.tx.up.sql

CREATE TABLE IF NOT EXISTS "tractors"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "primary_worker_id" TEXT NOT NULL,
    "equipment_type_id" TEXT NOT NULL,
    "equipment_manufacturer_id" TEXT NOT NULL,
    "state_id" TEXT,
    "fleet_code_id" TEXT,
    "secondary_worker_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Available',
    "code" TEXT NOT NULL,
    "model" TEXT,
    "make" TEXT,
    "year" INTEGER,
    "license_plate_number" TEXT,
    "registration_number" TEXT,
    "registration_expiry" INTEGER,
    "vin" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_tractors" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tractors_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_equipment_type" FOREIGN KEY ("equipment_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id", "business_unit_id", "organization_id") REFERENCES "equipment_manufacturers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_tractors_fleet_code" FOREIGN KEY ("fleet_code_id", "organization_id", "business_unit_id") REFERENCES "fleet_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_primary_worker" FOREIGN KEY ("primary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_secondary_worker" FOREIGN KEY ("secondary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_tractors_code" ON "tractors" (lower("code"), "organization_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_tractors_primary_worker_id" ON "tractors" ("primary_worker_id")WHERE
    "primary_worker_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tractors_status" ON "tractors" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tractors_business_unit_organization" ON "tractors" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tractors_created_updated" ON "tractors" ("created_at", "updated_at");
