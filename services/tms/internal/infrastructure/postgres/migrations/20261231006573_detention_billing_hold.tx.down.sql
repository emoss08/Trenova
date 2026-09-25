DROP INDEX IF EXISTS "idx_detention_occurrences_billing_hold";

--bun:split
DROP INDEX IF EXISTS "idx_invoice_lines_additional_charge";

--bun:split
ALTER TABLE "invoice_lines"
    DROP COLUMN IF EXISTS "additional_charge_id";
