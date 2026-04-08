ALTER TABLE billing_queue_items
    DROP COLUMN IF EXISTS "number";

--bun:split
DROP INDEX IF EXISTS "uq_billing_queue_items_number";

