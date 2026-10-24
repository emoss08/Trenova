--keel:notx
ALTER TYPE billing_queue_status
    ADD VALUE IF NOT EXISTS 'SentBackToOps';

--keel:split
ALTER TYPE billing_queue_status
    ADD VALUE IF NOT EXISTS 'OnHold';

--keel:split
ALTER TABLE billing_queue_items
    ADD COLUMN IF NOT EXISTS "exception_reason_code" varchar(50);

--keel:split
ALTER TABLE billing_queue_items
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_status";

--keel:split
ALTER TABLE billing_queue_items
    ADD CONSTRAINT "ck_billing_queue_items_status" CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception', 'SentBackToOps', 'OnHold'));

