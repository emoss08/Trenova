-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260121000000_custom_field_definitions.tx.up.sql

CREATE TABLE IF NOT EXISTS "custom_field_definitions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "field_type" TEXT NOT NULL,
    "is_required" INTEGER NOT NULL DEFAULT 0,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "color" TEXT,
    "options" TEXT,
    "validation_rules" TEXT,
    "default_value" TEXT,
    "ui_attributes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_custom_field_definitions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_custom_field_definitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_custom_field_definitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_custom_field_definitions_name" UNIQUE ("organization_id", "business_unit_id", "resource_type", "name")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfd_resource_type" ON "custom_field_definitions" ("resource_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfd_org_bu" ON "custom_field_definitions" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfd_active" ON "custom_field_definitions" ("is_active")WHERE is_active = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfd_resource_active" ON "custom_field_definitions" ("organization_id", "business_unit_id", "resource_type")WHERE is_active = TRUE;
