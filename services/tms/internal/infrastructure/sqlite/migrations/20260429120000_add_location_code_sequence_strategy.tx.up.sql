-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260429120000_add_location_code_sequence_strategy.tx.up.sql

ALTER TABLE "sequence_configs" ADD COLUMN "location_code_strategy" TEXT;

--bun:split

DROP INDEX IF EXISTS "idx_locations_search";
