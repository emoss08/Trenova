-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261002000000_formula_template_indexes.tx.up.sql
CREATE UNIQUE INDEX IF NOT EXISTS "uk_formula_templates_name" ON "formula_templates" ("organization_id", "business_unit_id", "name");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_tenant_created" ON "formula_templates" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_formula_template" ON "accessorial_charges" ("formula_template_id", "organization_id", "business_unit_id")
WHERE
    "formula_template_id" IS NOT NULL;
