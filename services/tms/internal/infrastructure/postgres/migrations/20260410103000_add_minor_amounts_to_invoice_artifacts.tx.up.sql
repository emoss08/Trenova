ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS subtotal_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS other_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS applied_amount_minor BIGINT NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE invoice_lines
    ADD COLUMN IF NOT EXISTS amount_minor BIGINT NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE invoice_adjustments
    ADD COLUMN IF NOT EXISTS credit_total_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rebill_total_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS net_delta_amount_minor BIGINT NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE invoice_adjustment_lines
    ADD COLUMN IF NOT EXISTS credit_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rebill_amount_minor BIGINT NOT NULL DEFAULT 0;

--bun:split
CREATE OR REPLACE FUNCTION bankers_round_minor(input NUMERIC)
RETURNS BIGINT
LANGUAGE SQL
IMMUTABLE
AS $$
    SELECT CASE
        WHEN input IS NULL THEN 0::BIGINT
        ELSE (
            CASE
                WHEN ABS(input * 100 - trunc(input * 100)) <> 0.5 THEN round(input * 100)
                WHEN MOD(ABS(trunc(input * 100))::BIGINT, 2) = 0 THEN trunc(input * 100)
                ELSE trunc(input * 100) + sign(input * 100)
            END
        )::BIGINT
    END;
$$;

--bun:split
UPDATE invoices
SET
    subtotal_amount_minor = bankers_round_minor(subtotal_amount),
    other_amount_minor = bankers_round_minor(other_amount),
    total_amount_minor = bankers_round_minor(total_amount),
    applied_amount_minor = bankers_round_minor(applied_amount);

--bun:split
UPDATE invoice_lines
SET amount_minor = bankers_round_minor(amount);

--bun:split
UPDATE invoice_adjustments
SET
    credit_total_amount_minor = bankers_round_minor(credit_total_amount),
    rebill_total_amount_minor = bankers_round_minor(rebill_total_amount),
    net_delta_amount_minor = bankers_round_minor(net_delta_amount);

--bun:split
UPDATE invoice_adjustment_lines
SET
    credit_amount_minor = bankers_round_minor(credit_amount),
    rebill_amount_minor = bankers_round_minor(rebill_amount);
