--keel:notx
-- A load tender is found by what a person types for it: its number as the partner sent it (the
-- 204's shipment identification), its BOL, PRO or PO number, or its status. The words are kept
-- as a stored search vector, written by the database on every insert and update and filled for
-- every existing tender when the column is added, so a search never reads the tender payloads.
ALTER TABLE "edi_load_tender_transfers"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("tender_payload" -> 'ratingDetail' ->> 'externalShipmentId', '')), 'A')
        || setweight(immutable_to_tsvector('simple', COALESCE("tender_payload" ->> 'bol', '')), 'A')
        || setweight(immutable_to_tsvector('simple', COALESCE("tender_payload" ->> 'proNumber', '') || ' ' || COALESCE("tender_payload" ->> 'poNumber', '')), 'B')
        || setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'C')
    ) STORED;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_edi_load_tender_transfers_search" ON "edi_load_tender_transfers" USING GIN("search_vector");
