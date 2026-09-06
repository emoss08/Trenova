-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261006000000_worker_pto_workflow.tx.up.sql

ALTER TABLE "worker_pto" ADD COLUMN "rejection_reason" TEXT;

--bun:split

ALTER TABLE "worker_pto" ADD COLUMN "cancellation_reason" TEXT;

--bun:split

ALTER TABLE "worker_pto" ADD COLUMN "cancelled_by_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_worker_status_dates"
    ON "worker_pto" ("organization_id", "business_unit_id", "worker_id", "status", "start_date", "end_date");
