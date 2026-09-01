-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261001000000_formula_template_rounding.tx.down.sql
ALTER TABLE "formula_template_versions"
DROP COLUMN "rounding_mode";

--bun:split
ALTER TABLE "formula_template_versions"
DROP COLUMN "rounding_precision";

--bun:split
ALTER TABLE "formula_templates"
DROP COLUMN "rounding_mode";

--bun:split
ALTER TABLE "formula_templates"
DROP COLUMN "rounding_precision";
