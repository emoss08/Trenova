-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241223200800_fleet_code.tx.up.sql

CREATE TABLE IF NOT EXISTS "fleet_codes"(
    "id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "manager_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "description" TEXT,
    "revenue_goal" REAL,
    "deadhead_goal" REAL,
    "mileage_goal" REAL,
    "color" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fleet_codes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fleet_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_manager" FOREIGN KEY ("manager_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fleet_codes_manager" ON "fleet_codes" ("manager_id")WHERE
    manager_id IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fleet_codes_color" ON "fleet_codes" ("color");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fleet_codes_created_updated" ON "fleet_codes" ("created_at", "updated_at");

--bun:split

ALTER TABLE "workers" ADD COLUMN "fleet_code_id" TEXT;
