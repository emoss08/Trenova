-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251020042421_add_distance_override.tx.up.sql

CREATE TABLE IF NOT EXISTS "distance_overrides"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "origin_location_id" TEXT NOT NULL,
    "destination_location_id" TEXT NOT NULL,
    "customer_id" TEXT,
    "distance" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_distance_overrides" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_overrides_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_distance_overrides_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_distance_overrides_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_distance_overrides_business_unit" ON "distance_overrides" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_distance_overrides_created_at" ON "distance_overrides" ("created_at", "updated_at");
