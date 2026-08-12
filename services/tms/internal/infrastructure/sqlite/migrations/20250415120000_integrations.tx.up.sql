-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250415120000_integrations.tx.up.sql

CREATE TABLE IF NOT EXISTS "integrations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "built_by" TEXT,
    "category" TEXT NOT NULL,
    "configuration" TEXT,
    "docs_url" TEXT,
    "featured" INTEGER NOT NULL DEFAULT 0,
    "logo_url" TEXT,
    "website_url" TEXT,
    "enabled_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_integrations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_integrations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_integrations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_integrations_enabled_by" FOREIGN KEY ("enabled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_integrations_organization_type" UNIQUE ("organization_id", "business_unit_id", "type")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_integrations_business_unit" ON "integrations" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_integrations_type" ON "integrations" ("type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_integrations_created_at" ON "integrations" ("created_at", "updated_at");
