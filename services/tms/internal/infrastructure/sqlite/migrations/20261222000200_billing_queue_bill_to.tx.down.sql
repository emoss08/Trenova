-- Hand-completed mirror of 20261222000200_billing_queue_bill_to.tx.down.sql.
-- SQLite cannot re-add the dropped CHECK constraint; the columns come off.

DROP INDEX IF EXISTS "idx_billing_queue_items_bill_to";

--bun:split

ALTER TABLE "billing_queue_items" DROP COLUMN "bill_to_customer_id";

--bun:split

ALTER TABLE "billing_queue_items" DROP COLUMN "allocated_total_amount";

--bun:split

ALTER TABLE "billing_queue_items" DROP COLUMN "allocated_total_amount_minor";
