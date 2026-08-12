-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260316103000_add_hold_reason_id_to_shipment_holds.tx.up.sql

ALTER TABLE "shipment_holds" ADD COLUMN "hold_reason_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_hold_reason" ON "shipment_holds" ("hold_reason_id");
