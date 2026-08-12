-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260118130000_formula_template_version_tags.tx.up.sql

ALTER TABLE "formula_template_versions" ADD COLUMN "tags" TEXT;
