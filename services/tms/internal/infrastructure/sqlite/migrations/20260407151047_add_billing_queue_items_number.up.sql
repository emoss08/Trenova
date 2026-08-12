-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260407151047_add_billing_queue_items_number.up.sql

ALTER TABLE "billing_queue_items" ADD COLUMN "number" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_queue_items_number" ON "billing_queue_items" ("number");
