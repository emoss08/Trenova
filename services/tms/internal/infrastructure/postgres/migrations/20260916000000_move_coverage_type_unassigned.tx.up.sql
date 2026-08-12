ALTER TABLE "shipment_moves"
    ALTER COLUMN "coverage_type" SET DEFAULT 'unassigned';

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
            AND "a"."archived_at" IS NULL);

--bun:split
DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves"("coverage_type", "organization_id")
WHERE
    "coverage_type" = 'carrier';

--bun:split
COMMENT ON COLUMN "shipment_moves"."coverage_type" IS 'How the move is covered: unassigned until a company driver (driver) or an external carrier (carrier) takes it';
