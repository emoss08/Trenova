-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260117074127_formula_template_versioning.tx.up.sql

ALTER TABLE "formula_templates" ADD COLUMN "source_template_id" TEXT;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "source_version_number" INTEGER;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "current_version_number" INTEGER NOT NULL DEFAULT 1;

--bun:split

CREATE TABLE IF NOT EXISTS "formula_template_versions"(
    "id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "version_number" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "type" TEXT NOT NULL,
    "expression" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "schema_id" TEXT NOT NULL,
    "variable_definitions" TEXT NOT NULL DEFAULT '[]',
    "metadata" TEXT,
    "change_message" TEXT,
    "change_summary" TEXT,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_formula_template_versions" PRIMARY KEY ("id"),
    CONSTRAINT "uk_formula_template_versions_template_version" UNIQUE ("template_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "fk_formula_template_versions_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_versions_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_versions_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_template" ON "formula_template_versions" ("template_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_created_at" ON "formula_template_versions" ("created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_templates_source" ON "formula_templates" ("source_template_id")WHERE "source_template_id" IS NOT NULL;
