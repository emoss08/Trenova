-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260829000000_edi_test_case_expected_codes.tx.up.sql

ALTER TABLE "edi_test_cases" ADD COLUMN "expected_warning_codes" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "edi_test_cases" ADD COLUMN "expected_error_codes" TEXT NOT NULL DEFAULT '[]';
