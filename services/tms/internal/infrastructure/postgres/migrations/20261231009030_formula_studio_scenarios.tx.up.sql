--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "formula_template_test_cases"
    ADD COLUMN IF NOT EXISTS "recorded_on_version" bigint,
    ADD COLUMN IF NOT EXISTS "recorded_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "recorded_at" bigint,
    ADD COLUMN IF NOT EXISTS "recorded_content" jsonb;

--bun:split
UPDATE "formula_template_test_cases"
SET "recorded_by_id" = "created_by_id",
    "recorded_at" = "created_at"
WHERE "recorded_by_id" IS NULL;

--bun:split
ALTER TABLE "formula_template_test_cases"
    ADD CONSTRAINT "fk_formula_template_test_cases_recorded_by" FOREIGN KEY ("recorded_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL;

--bun:split
ALTER TABLE "formula_template_test_cases"
    ADD CONSTRAINT "chk_formula_template_test_cases_recorded_on_version" CHECK (
        "recorded_on_version" IS NULL OR "recorded_on_version" >= 0
    );

--bun:split
ALTER TABLE "formula_template_reviews"
    ADD COLUMN IF NOT EXISTS "requested_reviewer_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "reviewer_note" text;

--bun:split
ALTER TABLE "formula_template_reviews"
    ADD CONSTRAINT "fk_formula_template_reviews_requested_reviewer" FOREIGN KEY ("requested_reviewer_id")
        REFERENCES "users"("id") ON DELETE SET NULL;

--bun:split
ALTER TABLE "formula_template_reviews"
    ADD CONSTRAINT "chk_formula_template_reviews_reviewer_note_length" CHECK (
        "reviewer_note" IS NULL OR char_length("reviewer_note") <= 4000
    );
