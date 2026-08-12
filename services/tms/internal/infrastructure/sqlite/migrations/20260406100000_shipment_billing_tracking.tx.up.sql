-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260406100000_shipment_billing_tracking.tx.up.sql

ALTER TABLE "shipments" ADD COLUMN "billing_transfer_status" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "transferred_to_billing_at" INTEGER;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "marked_ready_to_bill_at" INTEGER;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "billed_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipments_billing_transfer_status
    ON shipments ("billing_transfer_status")WHERE "billing_transfer_status" IS NOT NULL;
