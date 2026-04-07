--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS "billing_transfer_status" varchar(50),
    ADD COLUMN IF NOT EXISTS "transferred_to_billing_at" bigint,
    ADD COLUMN IF NOT EXISTS "marked_ready_to_bill_at" bigint,
    ADD COLUMN IF NOT EXISTS "billed_at" bigint;

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_billing_transfer_status
    ON shipments("billing_transfer_status")
    WHERE "billing_transfer_status" IS NOT NULL;
