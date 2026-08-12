-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260406000000_billing_queue_v2.up.sql

ALTER TABLE "billing_queue_items" ADD COLUMN "exception_reason_code" TEXT;
