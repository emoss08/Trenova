ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "recheck_due_at" bigint;

COMMENT ON COLUMN "billing_queue_items"."recheck_due_at" IS 'When the item''s billing checks are next due to run; null when nothing is waiting. Set in the transaction that changed the item, cleared by the tenant''s drain once the checks ran';
