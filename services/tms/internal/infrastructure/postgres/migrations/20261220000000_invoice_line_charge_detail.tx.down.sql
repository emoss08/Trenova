DROP INDEX IF EXISTS "idx_invoice_lines_accessorial_charge";

--bun:split
ALTER TABLE "invoice_lines"
    DROP COLUMN IF EXISTS "formula_template_name",
    DROP COLUMN IF EXISTS "rate_basis_amount",
    DROP COLUMN IF EXISTS "rate",
    DROP COLUMN IF EXISTS "rate_unit",
    DROP COLUMN IF EXISTS "charge_method",
    DROP COLUMN IF EXISTS "charge_code",
    DROP COLUMN IF EXISTS "accessorial_charge_id";
