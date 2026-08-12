-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250206004201_assignment.tx.up.sql

CREATE TABLE IF NOT EXISTS "assignments"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "primary_worker_id" TEXT NOT NULL,
    "tractor_id" TEXT NOT NULL,
    "trailer_id" TEXT,
    "secondary_worker_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'New',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_primary_worker" FOREIGN KEY ("primary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_secondary_worker" FOREIGN KEY ("secondary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_tractor" FOREIGN KEY ("tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_trailer" FOREIGN KEY ("trailer_id", "organization_id", "business_unit_id") REFERENCES "trailers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_status" ON "assignments" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_created_at" ON "assignments" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_business_unit" ON "assignments" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_shipment_move" ON "assignments" ("shipment_move_id", "organization_id");
