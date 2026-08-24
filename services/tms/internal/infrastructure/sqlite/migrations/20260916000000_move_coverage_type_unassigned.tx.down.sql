-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260916000000_move_coverage_type_unassigned.tx.down.sql

DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split

UPDATE
    "shipment_moves"
SET
    "coverage_type" = 'driver'
WHERE
    "coverage_type" = 'unassigned';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves" ("coverage_type", "organization_id")WHERE
    "coverage_type" <> 'driver';
