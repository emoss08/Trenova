--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_formula_templates_in_review_since";

--bun:split
DROP TABLE IF EXISTS "formula_template_reviews";

--bun:split
DROP TYPE IF EXISTS "formula_template_review_decision_enum";
