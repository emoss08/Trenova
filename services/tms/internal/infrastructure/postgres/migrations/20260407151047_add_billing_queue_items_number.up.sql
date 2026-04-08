ALTER TABLE billing_queue_items
    ADD COLUMN IF NOT EXISTS "number" varchar(100);

--bun:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_billing_queue_items_number" ON "billing_queue_items"("number");

