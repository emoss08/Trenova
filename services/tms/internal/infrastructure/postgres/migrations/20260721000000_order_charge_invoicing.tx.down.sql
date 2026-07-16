--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "order_charges"
    DROP CONSTRAINT IF EXISTS "fk_order_charges_invoice";

--bun:split
DROP INDEX IF EXISTS idx_order_charges_invoice;

--bun:split
ALTER TABLE "order_charges"
    DROP COLUMN IF EXISTS "invoice_id",
    DROP COLUMN IF EXISTS "invoiced_at";
