DROP TABLE IF EXISTS "carrier_assignment_accessorials";

--bun:split
DROP TABLE IF EXISTS "carrier_assignments";

--bun:split
DROP INDEX IF EXISTS "idx_shipment_moves_coverage_type";

--bun:split
ALTER TABLE "shipment_moves"
    DROP COLUMN IF EXISTS "coverage_type";
