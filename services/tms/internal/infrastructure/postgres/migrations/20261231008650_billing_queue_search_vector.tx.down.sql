DROP INDEX IF EXISTS "idx_billing_queue_items_search";

--bun:split

ALTER TABLE "billing_queue_items"
    DROP COLUMN IF EXISTS search_vector;
