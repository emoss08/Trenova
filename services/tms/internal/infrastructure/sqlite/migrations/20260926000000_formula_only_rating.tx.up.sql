-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260926000000_formula_only_rating.tx.up.sql

ALTER TABLE "rate_matrices" ADD COLUMN "formula_template_id" TEXT;

--bun:split

ALTER TABLE "rate_agreement_rules" DROP COLUMN "rating_basis";

--bun:split

ALTER TABLE "rate_agreement_rules" DROP COLUMN "percent_basis";

--bun:split

ALTER TABLE "rate_matrices" DROP COLUMN "value_kind";
