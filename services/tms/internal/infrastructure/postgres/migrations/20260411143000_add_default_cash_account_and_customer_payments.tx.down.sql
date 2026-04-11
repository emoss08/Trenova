DROP INDEX IF EXISTS idx_customer_payment_applications_invoice;

--bun:split
DROP INDEX IF EXISTS idx_customer_payments_customer_date;

--bun:split
DROP TABLE IF EXISTS customer_payment_applications;

--bun:split
DROP TABLE IF EXISTS customer_payments;

--bun:split
ALTER TABLE accounting_controls
    DROP COLUMN IF EXISTS default_cash_account_id;
