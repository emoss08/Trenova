-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250711022004_add_formula_template.tx.up.sql

CREATE TABLE IF NOT EXISTS "formula_templates"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "type" TEXT NOT NULL DEFAULT 'FreightCharge',
    "expression" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "schema_id" TEXT NOT NULL DEFAULT 'shipment',
    "variable_definitions" TEXT NOT NULL DEFAULT '[]',
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_formula_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_formula_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_formula_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_templates_type_status" ON "formula_templates" ("type", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_templates_bu_org" ON "formula_templates" ("business_unit_id", "organization_id");

--bun:split

ALTER TABLE "shipments" ADD COLUMN "formula_template_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_formula_template" ON "shipments" ("formula_template_id", "business_unit_id", "organization_id")WHERE
    "formula_template_id" IS NOT NULL;
