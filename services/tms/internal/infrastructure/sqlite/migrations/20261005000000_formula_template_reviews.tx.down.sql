-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261005000000_formula_template_reviews.tx.down.sql

DROP INDEX IF EXISTS "idx_formula_templates_in_review_since";

--bun:split
DROP TABLE IF EXISTS "formula_template_reviews";
