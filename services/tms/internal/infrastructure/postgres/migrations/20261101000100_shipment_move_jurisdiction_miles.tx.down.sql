--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE IF EXISTS "distance_controls"
    DROP COLUMN IF EXISTS "capture_jurisdiction_miles";

--bun:split
ALTER TABLE IF EXISTS "stored_mileages"
    DROP COLUMN IF EXISTS "jurisdiction_distances";

--bun:split
DROP INDEX IF EXISTS "idx_shipment_move_jurisdiction_miles_shipment";

--bun:split
DROP INDEX IF EXISTS "idx_shipment_move_jurisdiction_miles_jurisdiction";

--bun:split
DROP INDEX IF EXISTS "uq_shipment_move_jurisdiction_miles_move_jurisdiction";

--bun:split
DROP TABLE IF EXISTS "shipment_move_jurisdiction_miles";
