-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250127225614_customer.tx.up.sql

CREATE TABLE IF NOT EXISTS "customers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT,
    "address_line_1" TEXT,
    "address_line_2" TEXT,
    "city" TEXT,
    "postal_code" TEXT NOT NULL,
    "is_geocoded" INTEGER NOT NULL DEFAULT 0,
    "longitude" TEXT,
    "latitude" TEXT,
    "place_id" TEXT,
    "external_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_customers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_customers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_customers_code" ON "customers" (lower("code"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customers_name" ON "customers" ("name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customers_business_unit_organization" ON "customers" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customers_created_updated" ON "customers" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customers_status" ON "customers" ("status");

--bun:split

ALTER TABLE "shipments" ADD COLUMN "customer_id" TEXT NOT NULL;
