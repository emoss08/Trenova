ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "exchange_rate_date" bigint;

ALTER TABLE "customer_payments"
    ADD COLUMN IF NOT EXISTS "exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "exchange_rate_date" bigint;

ALTER TABLE "carrier_settlements"
    ADD COLUMN IF NOT EXISTS "exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "exchange_rate_date" bigint,
    ADD COLUMN IF NOT EXISTS "paid_exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "paid_exchange_rate_date" bigint;

ALTER TABLE "driver_settlements"
    ADD COLUMN IF NOT EXISTS "exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "exchange_rate_date" bigint,
    ADD COLUMN IF NOT EXISTS "paid_exchange_rate" numeric(24,12),
    ADD COLUMN IF NOT EXISTS "paid_exchange_rate_date" bigint;
