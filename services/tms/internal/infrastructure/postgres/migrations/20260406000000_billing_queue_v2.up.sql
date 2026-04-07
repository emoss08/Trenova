--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TYPE billing_queue_status ADD VALUE IF NOT EXISTS 'SentBackToOps';

--bun:split
ALTER TYPE billing_queue_status ADD VALUE IF NOT EXISTS 'OnHold';

--bun:split
ALTER TABLE billing_queue_items
    ADD COLUMN IF NOT EXISTS "exception_reason_code" varchar(50);

--bun:split
ALTER TABLE billing_queue_items
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_status";

--bun:split
ALTER TABLE billing_queue_items
    ADD CONSTRAINT "ck_billing_queue_items_status"
    CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception', 'SentBackToOps', 'OnHold'));
