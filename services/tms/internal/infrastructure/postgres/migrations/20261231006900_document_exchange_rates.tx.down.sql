ALTER TABLE "driver_settlements"
    DROP COLUMN IF EXISTS "paid_exchange_rate_date",
    DROP COLUMN IF EXISTS "paid_exchange_rate",
    DROP COLUMN IF EXISTS "exchange_rate_date",
    DROP COLUMN IF EXISTS "exchange_rate";

ALTER TABLE "carrier_settlements"
    DROP COLUMN IF EXISTS "paid_exchange_rate_date",
    DROP COLUMN IF EXISTS "paid_exchange_rate",
    DROP COLUMN IF EXISTS "exchange_rate_date",
    DROP COLUMN IF EXISTS "exchange_rate";

ALTER TABLE "customer_payments"
    DROP COLUMN IF EXISTS "exchange_rate_date",
    DROP COLUMN IF EXISTS "exchange_rate";

ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "exchange_rate_date",
    DROP COLUMN IF EXISTS "exchange_rate";
