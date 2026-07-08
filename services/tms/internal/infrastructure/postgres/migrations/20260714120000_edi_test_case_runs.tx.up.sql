ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_at" bigint;

--bun:split
ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_passed" boolean;

--bun:split
ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_warnings" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_errors" integer NOT NULL DEFAULT 0;
