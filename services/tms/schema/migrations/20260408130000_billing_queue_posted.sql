--keel:notx
ALTER TYPE billing_queue_status
    ADD VALUE IF NOT EXISTS 'Posted';

--keel:split
ALTER TABLE billing_queue_items
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_status";

--keel:split
ALTER TABLE billing_queue_items
    ADD CONSTRAINT "ck_billing_queue_items_status"
    CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Posted', 'Canceled', 'Exception', 'SentBackToOps', 'OnHold'));
