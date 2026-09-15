DROP INDEX IF EXISTS "idx_invoice_lines_charge_allocation";

--bun:split
ALTER TABLE "invoice_lines"
    DROP COLUMN IF EXISTS "allocation_percent",
    DROP COLUMN IF EXISTS "charge_allocation_id";

--bun:split
DROP INDEX IF EXISTS "idx_invoices_shipment";

--bun:split
DROP INDEX IF EXISTS "idx_invoices_shipper_customer";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_shipper_customer";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "shipper_customer_id",
    DROP COLUMN IF EXISTS "is_split_bill";
