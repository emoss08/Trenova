-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250126210710_trailer.tx.up.sql

CREATE TABLE IF NOT EXISTS "trailers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "equipment_type_id" TEXT NOT NULL,
    "equipment_manufacturer_id" TEXT NOT NULL,
    "registration_state_id" TEXT,
    "fleet_code_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Available',
    "code" TEXT NOT NULL,
    "model" TEXT,
    "make" TEXT,
    "year" INTEGER,
    "license_plate_number" TEXT,
    "vin" TEXT,
    "registration_number" TEXT,
    "max_load_weight" INTEGER,
    "last_inspection_date" INTEGER,
    "registration_expiry" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_trailers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_trailers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_equipment_type" FOREIGN KEY ("equipment_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id", "business_unit_id", "organization_id") REFERENCES "equipment_manufacturers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_registration_state" FOREIGN KEY ("registration_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_trailers_fleet_code" FOREIGN KEY ("fleet_code_id", "organization_id", "business_unit_id") REFERENCES "fleet_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_trailers_code" ON "trailers" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_trailers_status" ON "trailers" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_trailers_business_unit_organization" ON "trailers" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_trailers_created_updated" ON "trailers" ("created_at", "updated_at");
