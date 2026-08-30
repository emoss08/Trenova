-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260928100000_shipment_auto_rating.tx.down.sql

DROP INDEX IF EXISTS "idx_shipments_auto_rated";

--bun:split

ALTER TABLE "shipments" DROP COLUMN "auto_rated_at";

--bun:split

ALTER TABLE "shipments" DROP COLUMN "auto_rated";
