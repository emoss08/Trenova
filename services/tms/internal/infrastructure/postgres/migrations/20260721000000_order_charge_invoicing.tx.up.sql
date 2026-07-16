--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Order charges are billed exactly once: the first grouped invoice for the order
-- carries them and stamps invoice_id/invoiced_at so later passes (partial-order
-- invoicing, rebills) never duplicate them.
ALTER TABLE "order_charges"
    ADD COLUMN IF NOT EXISTS "invoice_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "invoiced_at" bigint;

--bun:split
ALTER TABLE "order_charges"
    ADD CONSTRAINT "fk_order_charges_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_order_charges_invoice ON order_charges("invoice_id")
WHERE
    "invoice_id" IS NOT NULL;
