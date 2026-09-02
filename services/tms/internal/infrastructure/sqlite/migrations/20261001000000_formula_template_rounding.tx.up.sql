-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261001000000_formula_template_rounding.tx.up.sql
ALTER TABLE "formula_templates"
ADD COLUMN "rounding_mode" TEXT NOT NULL DEFAULT 'HalfUp';

--bun:split
ALTER TABLE "formula_templates"
ADD COLUMN "rounding_precision" INTEGER NOT NULL DEFAULT 2 CHECK ("rounding_precision" BETWEEN 0 AND 4);

--bun:split
ALTER TABLE "formula_template_versions"
ADD COLUMN "rounding_mode" TEXT NOT NULL DEFAULT 'HalfUp';

--bun:split
ALTER TABLE "formula_template_versions"
ADD COLUMN "rounding_precision" INTEGER NOT NULL DEFAULT 2 CHECK ("rounding_precision" BETWEEN 0 AND 4);
