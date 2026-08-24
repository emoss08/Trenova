ALTER TABLE "shipment_moves"
    ALTER COLUMN "coverage_type" SET DEFAULT 'unassigned';

--bun:split
-- A move labelled 'driver' that no live driver assignment backs but an active
-- carrier assignment does is carrier coverage wearing the old column default.
-- Relabel it before the 'unassigned' sweep so that coverage is never lost.
-- Inactive is 'Canceled' for carrier assignments (they have no archived_at) and
-- archived_at for driver assignments.
UPDATE
    "shipment_moves" AS "sm"
SET
    "coverage_type" = 'carrier'
WHERE
    "sm"."coverage_type" = 'driver'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "assignments" AS "a"
        WHERE
            "a"."shipment_move_id" = "sm"."id"
            AND "a"."organization_id" = "sm"."organization_id"
            AND "a"."business_unit_id" = "sm"."business_unit_id"
            AND "a"."archived_at" IS NULL)
    AND EXISTS (
        SELECT
            1
        FROM
            "carrier_assignments" AS "ca"
        WHERE
            "ca"."shipment_move_id" = "sm"."id"
            AND "ca"."organization_id" = "sm"."organization_id"
            AND "ca"."business_unit_id" = "sm"."business_unit_id"
            AND "ca"."status" <> 'Canceled');

--bun:split
UPDATE
    "shipment_moves" AS "sm"
SET
    "coverage_type" = 'unassigned'
WHERE
    "sm"."coverage_type" = 'driver'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "assignments" AS "a"
        WHERE
            "a"."shipment_move_id" = "sm"."id"
            AND "a"."organization_id" = "sm"."organization_id"
            AND "a"."business_unit_id" = "sm"."business_unit_id"
            AND "a"."archived_at" IS NULL)
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "carrier_assignments" AS "ca"
        WHERE
            "ca"."shipment_move_id" = "sm"."id"
            AND "ca"."organization_id" = "sm"."organization_id"
            AND "ca"."business_unit_id" = "sm"."business_unit_id"
            AND "ca"."status" <> 'Canceled');

--bun:split
DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves"("coverage_type", "organization_id")
WHERE
    "coverage_type" = 'carrier';

--bun:split
COMMENT ON COLUMN "shipment_moves"."coverage_type" IS 'How the move is covered: unassigned until a company driver (driver) or an external carrier (carrier) takes it';
