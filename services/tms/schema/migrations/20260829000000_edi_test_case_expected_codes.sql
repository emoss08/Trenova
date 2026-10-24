ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "expected_warning_codes" jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "expected_error_codes" jsonb NOT NULL DEFAULT '[]'::jsonb;
