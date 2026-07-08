ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "last_run_at";

--bun:split
ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "last_run_passed";

--bun:split
ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "last_run_warnings";

--bun:split
ALTER TABLE "edi_test_cases"
    DROP COLUMN IF EXISTS "last_run_errors";
