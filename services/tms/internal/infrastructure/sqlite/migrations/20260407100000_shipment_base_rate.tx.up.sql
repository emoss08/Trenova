-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260407100000_shipment_base_rate.tx.up.sql

ALTER TABLE "shipments" ADD COLUMN "base_rate" NUMERIC NOT NULL DEFAULT 0;

--bun:split

UPDATE shipments SET base_rate = freight_charge_amount WHERE base_rate = 0 AND freight_charge_amount > 0;
