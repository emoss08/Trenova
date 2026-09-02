-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261002000000_formula_template_indexes.tx.down.sql
DROP INDEX IF EXISTS "idx_rate_agreement_accessorials_formula_template";

--bun:split
DROP INDEX IF EXISTS "idx_rate_agreement_rules_formula_template";

--bun:split
DROP INDEX IF EXISTS "idx_rate_matrices_formula_template";

--bun:split
DROP INDEX IF EXISTS "idx_formula_templates_tenant_created";

--bun:split
DROP INDEX IF EXISTS "uk_formula_templates_name";
