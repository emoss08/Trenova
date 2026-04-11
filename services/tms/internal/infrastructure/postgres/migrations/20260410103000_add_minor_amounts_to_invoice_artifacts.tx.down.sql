ALTER TABLE invoice_adjustment_lines
    DROP COLUMN IF EXISTS rebill_amount_minor,
    DROP COLUMN IF EXISTS credit_amount_minor;

--bun:split
ALTER TABLE invoice_adjustments
    DROP COLUMN IF EXISTS net_delta_amount_minor,
    DROP COLUMN IF EXISTS rebill_total_amount_minor,
    DROP COLUMN IF EXISTS credit_total_amount_minor;

--bun:split
ALTER TABLE invoice_lines
    DROP COLUMN IF EXISTS amount_minor;

--bun:split
ALTER TABLE invoices
    DROP COLUMN IF EXISTS applied_amount_minor,
    DROP COLUMN IF EXISTS total_amount_minor,
    DROP COLUMN IF EXISTS other_amount_minor,
    DROP COLUMN IF EXISTS subtotal_amount_minor;

--bun:split
DROP FUNCTION IF EXISTS bankers_round_minor(NUMERIC);
