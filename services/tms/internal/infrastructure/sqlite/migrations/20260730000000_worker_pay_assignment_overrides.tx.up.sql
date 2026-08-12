-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260730000000_worker_pay_assignment_overrides.tx.up.sql

ALTER TABLE "worker_pay_assignments" ADD COLUMN "rate_overrides" TEXT;
