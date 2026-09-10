--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "fuel_purchase_import_batches"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("file_name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("feed_reference", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_purchase_import_batches_search" ON "fuel_purchase_import_batches" USING GIN(search_vector);

--bun:split
COMMENT ON COLUMN fuel_purchase_import_batches.search_vector IS 'What somebody would type to find a run: the uploaded file name, or the remote path a feed read.';
