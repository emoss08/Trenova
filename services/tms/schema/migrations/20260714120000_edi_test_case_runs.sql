ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_at" bigint;

ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_passed" boolean;

ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_warnings" integer NOT NULL DEFAULT 0;

ALTER TABLE "edi_test_cases"
    ADD COLUMN IF NOT EXISTS "last_run_errors" integer NOT NULL DEFAULT 0;
