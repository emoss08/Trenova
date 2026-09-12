--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP INDEX IF EXISTS idx_billing_queue_items_invoice;

--bun:split
ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "fk_billing_queue_items_invoice";

--bun:split
ALTER TABLE "billing_queue_items"
    DROP COLUMN IF EXISTS "invoice_id";
