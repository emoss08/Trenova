-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260407000000_shipment_rating_detail.tx.up.sql

ALTER TABLE "shipments" ADD COLUMN "rating_detail" TEXT;
