--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_fuel_purchase_import_batches_search";

--bun:split
ALTER TABLE "fuel_purchase_import_batches"
    DROP COLUMN IF EXISTS "search_vector";
