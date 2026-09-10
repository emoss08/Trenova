DROP INDEX IF EXISTS "idx_fuel_purchase_import_batches_search";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP COLUMN IF EXISTS "search_vector";
