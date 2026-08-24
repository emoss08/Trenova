DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split
UPDATE
    "shipment_moves"
SET
    "coverage_type" = 'driver'
WHERE
    "coverage_type" = 'unassigned';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves"("coverage_type", "organization_id")
WHERE
    "coverage_type" <> 'driver';

--bun:split
ALTER TABLE "shipment_moves"
    ALTER COLUMN "coverage_type" SET DEFAULT 'driver';
