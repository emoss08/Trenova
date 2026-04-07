--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE billing_queue_items
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_status";

--bun:split
ALTER TABLE billing_queue_items
    ADD CONSTRAINT "ck_billing_queue_items_status"
    CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception'));

--bun:split
ALTER TABLE billing_queue_items
    DROP COLUMN IF EXISTS "exception_reason_code";
