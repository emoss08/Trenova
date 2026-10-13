--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "formula_template_reviews"
    DROP CONSTRAINT IF EXISTS "chk_formula_template_reviews_reviewer_note_length",
    DROP CONSTRAINT IF EXISTS "fk_formula_template_reviews_requested_reviewer",
    DROP COLUMN IF EXISTS "reviewer_note",
    DROP COLUMN IF EXISTS "requested_reviewer_id";

--bun:split
ALTER TABLE "formula_template_test_cases"
    DROP CONSTRAINT IF EXISTS "chk_formula_template_test_cases_recorded_on_version",
    DROP CONSTRAINT IF EXISTS "fk_formula_template_test_cases_recorded_by",
    DROP COLUMN IF EXISTS "recorded_content",
    DROP COLUMN IF EXISTS "recorded_at",
    DROP COLUMN IF EXISTS "recorded_by_id",
    DROP COLUMN IF EXISTS "recorded_on_version";
