DROP TABLE IF EXISTS "exchange_rates";

--bun:split
-- Note: ALTER TYPE ... REMOVE VALUE is not supported in PostgreSQL.
-- The ExchangeRateAPI and FinancialData enum values cannot be removed via migration.
-- They will remain in the enum definition but become unused.
