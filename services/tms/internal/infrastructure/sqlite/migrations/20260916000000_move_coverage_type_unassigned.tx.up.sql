-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260916000000_move_coverage_type_unassigned.tx.up.sql
--
-- Hand-completed: the converter drops both statements this migration needs.
-- ALTER COLUMN ... SET DEFAULT has no SQLite equivalent and rebuilding
-- "shipment_moves" is not an option here because it is the foreign key target
-- of "assignments", "stops" and "carrier_assignments" and PRAGMA foreign_keys
-- cannot be toggled inside the transaction this migration runs in. The Go model
-- stamps coverage_type on every insert, so the column default is never
-- exercised. The correlated backfill is written out by hand because the
-- converter refuses any statement containing a subquery.

UPDATE
    "shipment_moves"
SET
    "coverage_type" = 'unassigned'
WHERE
    "coverage_type" = 'driver'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "assignments" AS "a"
        WHERE
            "a"."shipment_move_id" = "shipment_moves"."id"
            AND "a"."organization_id" = "shipment_moves"."organization_id"
            AND "a"."business_unit_id" = "shipment_moves"."business_unit_id"
            AND "a"."archived_at" IS NULL);

--bun:split

DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves" ("coverage_type", "organization_id")WHERE
    "coverage_type" = 'carrier';
