-- An invoice's customer is always the payer. When a shipment is billed to a
-- party other than the customer who ordered it, the shipper is recorded so the
-- invoice can say on whose behalf it bills. Lines carry the share they bill.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "shipper_customer_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "is_split_bill" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_shipper_customer" FOREIGN KEY ("shipper_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_shipper_customer" ON "invoices"("shipper_customer_id", "organization_id", "business_unit_id")
WHERE
    "shipper_customer_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_shipment" ON "invoices"("shipment_id", "organization_id", "business_unit_id")
WHERE
    "shipment_id" IS NOT NULL;

--bun:split
ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "allocation_percent" numeric(9, 6),
    ADD COLUMN IF NOT EXISTS "charge_allocation_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_lines_charge_allocation" ON "invoice_lines"("charge_allocation_id", "organization_id", "business_unit_id")
WHERE
    "charge_allocation_id" IS NOT NULL;
