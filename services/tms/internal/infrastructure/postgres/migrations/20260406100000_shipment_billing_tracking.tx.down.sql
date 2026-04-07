--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS idx_shipments_billing_transfer_status;

--bun:split
ALTER TABLE shipments
    DROP COLUMN IF EXISTS "billing_transfer_status",
    DROP COLUMN IF EXISTS "transferred_to_billing_at",
    DROP COLUMN IF EXISTS "marked_ready_to_bill_at",
    DROP COLUMN IF EXISTS "billed_at";
