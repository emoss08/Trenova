-- BillingQueueItem declares UseSearchVector, so a text search on the queue reads
-- bqi.search_vector; the column was never created, and every search failed.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("review_notes", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("exception_notes", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("cancel_reason", '')), 'B')
    ) STORED;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_queue_items_search" ON "billing_queue_items" USING GIN(search_vector);
