-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260930000000_formula_template_test_cases.tx.up.sql

CREATE TABLE IF NOT EXISTS "formula_template_test_cases"(
    "id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "variables" TEXT NOT NULL DEFAULT '{}',
    "expected_amount" REAL NOT NULL,
    "tolerance" REAL NOT NULL DEFAULT 0.01,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_formula_template_test_cases" PRIMARY KEY ("id"),
    CONSTRAINT "uk_formula_template_test_cases_name" UNIQUE ("template_id", "organization_id", "business_unit_id", "name"),
    CONSTRAINT "fk_formula_template_test_cases_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_test_cases_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_test_cases_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_template_test_cases_template" ON "formula_template_test_cases" ("template_id", "organization_id", "business_unit_id");
