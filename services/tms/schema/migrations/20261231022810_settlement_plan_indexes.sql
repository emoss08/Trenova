--keel:notx
-- Driver settlements, driver pay events, IFTA mileage and fuel purchases page newest first
-- (fuel purchases also within one tractor), and a fuel card's purchases page by when they were
-- bought: each order comes from an index instead of a sort of the tenant's rows. A card's import
-- rows still held for review page newest first, a purchase finds the row it was committed from,
-- and a batch's rows of one status page by row number. The card and batch indexes replace the
-- ones that found the rows but not their order.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlements_created" ON "driver_settlements"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_pay_events_created" ON "driver_pay_events"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ifta_jurisdiction_mileage_entries_traveled" ON "ifta_jurisdiction_mileage_entries"("organization_id", "business_unit_id", "traveled_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchases_created" ON "fuel_purchases"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchases_tractor_created" ON "fuel_purchases"("organization_id", "business_unit_id", "tractor_id", "created_at" DESC, "id" DESC);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchases_fuel_card_purchased" ON "fuel_purchases"("organization_id", "business_unit_id", "fuel_card_id", "purchased_at" DESC, "id" DESC)
WHERE
    "fuel_card_id" IS NOT NULL;

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_fuel_purchases_fuel_card";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchase_import_rows_held_card" ON "fuel_purchase_import_rows"("organization_id", "business_unit_id", "resolved_fuel_card_id", "created_at" DESC, "row_number")
WHERE
    "fuel_purchase_id" IS NULL AND "status" IN ('New', 'Error');

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchase_import_rows_purchase" ON "fuel_purchase_import_rows"("organization_id", "business_unit_id", "fuel_purchase_id", "created_at" DESC)
WHERE
    "fuel_purchase_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_fuel_purchase_import_rows_batch_status" ON "fuel_purchase_import_rows"("organization_id", "business_unit_id", "import_batch_id", "status", "row_number");

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_fuel_purchase_import_rows_status";
