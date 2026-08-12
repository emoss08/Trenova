-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260714120000_edi_test_case_runs.tx.up.sql

ALTER TABLE "edi_test_cases" ADD COLUMN "last_run_at" INTEGER;

--bun:split

ALTER TABLE "edi_test_cases" ADD COLUMN "last_run_passed" INTEGER;

--bun:split

ALTER TABLE "edi_test_cases" ADD COLUMN "last_run_warnings" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "edi_test_cases" ADD COLUMN "last_run_errors" INTEGER NOT NULL DEFAULT 0;
