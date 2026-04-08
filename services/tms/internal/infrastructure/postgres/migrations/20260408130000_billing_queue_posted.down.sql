ALTER TABLE billing_queue_items
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_status";

--bun:split
ALTER TABLE billing_queue_items
    ADD CONSTRAINT "ck_billing_queue_items_status"
    CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception', 'SentBackToOps', 'OnHold'));
