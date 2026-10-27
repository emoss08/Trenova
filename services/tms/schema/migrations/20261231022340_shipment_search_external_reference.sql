-- A shipment's search words match its external reference (the customer's own number for the
-- load) beside the pro number, BOL and status. The generated column is rebuilt with it, and its
-- GIN index with the column.
DROP INDEX IF EXISTS "idx_shipments_search";

ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "search_vector";

ALTER TABLE "shipments"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("pro_number", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("bol", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("external_reference", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'B')
    ) STORED;

CREATE INDEX IF NOT EXISTS "idx_shipments_search" ON "shipments" USING GIN ("search_vector");
