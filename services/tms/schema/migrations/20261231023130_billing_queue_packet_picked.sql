-- An item's packet counted as the biller's pick only while it named a document,
-- so taking the last one out sent the customer's required documents again. The
-- item now says outright whether the biller has picked: a pick may be empty, and
-- only an item with no pick falls back to the required documents.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "packet_picked" boolean NOT NULL DEFAULT false;

UPDATE "billing_queue_items"
SET "packet_picked" = true
WHERE jsonb_array_length("packet_document_ids") > 0
    AND NOT "packet_picked";
