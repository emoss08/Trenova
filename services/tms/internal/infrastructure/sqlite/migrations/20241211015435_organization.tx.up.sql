-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015435_organization.tx.up.sql

CREATE TABLE IF NOT EXISTS "organizations"(
    "id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "scac_code" TEXT NOT NULL,
    "dot_number" TEXT NOT NULL,
    "logo_url" TEXT,
    "bucket_name" TEXT NOT NULL,
    "address_line1" TEXT NOT NULL,
    "address_line2" TEXT,
    "city" TEXT NOT NULL,
    "postal_code" TEXT NOT NULL,
    "timezone" TEXT NOT NULL DEFAULT 'America/New_York',
    "tax_id" TEXT,
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_organizations_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_organizations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_name_business_unit" ON "organizations" (lower("name"), "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_scac_business_unit" ON "organizations" ("scac_code", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_dot_business_unit" ON "organizations" ("dot_number", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_organizations_business_unit" ON "organizations" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_organizations_state" ON "organizations" ("state_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_organizations_created_updated" ON "organizations" ("created_at", "updated_at");
