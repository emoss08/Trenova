-- An invoice line names the shipment charge it bills, so the detention
-- occurrences behind that charge are marked billed by exactly the invoices that
-- carry it, and released only once none of them does. Lines written before this
-- point carry no charge and are left as they are: nothing marked an occurrence
-- billed from them.
ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "additional_charge_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_lines_additional_charge" ON "invoice_lines"("additional_charge_id", "organization_id", "business_unit_id")
WHERE
    "additional_charge_id" IS NOT NULL;

--bun:split
-- The charges waiting on an approver, read by every approval and invoice path
-- for the shipments they bill and by the detention desk for the whole tenant.
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_billing_hold" ON "detention_occurrences"("organization_id", "business_unit_id", "shipment_id")
WHERE
    "status" = 'Pending'
    AND "requires_approval";
