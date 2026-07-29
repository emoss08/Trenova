ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "expected_warning_codes";

--bun:split
ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "expected_error_codes";
