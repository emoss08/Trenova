-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261101000200_tractor_fuel_type.tx.up.sql

ALTER TABLE "tractors" ADD COLUMN "fuel_type" TEXT NOT NULL DEFAULT 'Diesel';

--bun:split

ALTER TABLE "tractors" ADD COLUMN "ifta_qualified" INTEGER NOT NULL DEFAULT 1;
