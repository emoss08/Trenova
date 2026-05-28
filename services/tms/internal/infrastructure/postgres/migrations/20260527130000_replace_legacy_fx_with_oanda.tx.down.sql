DROP TABLE IF EXISTS "exchange_rate_settlement_quotes";

--bun:split
DROP INDEX IF EXISTS idx_exchange_rates_lookup;

--bun:split
ALTER TABLE exchange_rates
    DROP CONSTRAINT IF EXISTS uq_exchange_rates_org_currency_date,
    DROP CONSTRAINT IF EXISTS chk_exchange_rates_provider,
    DROP CONSTRAINT IF EXISTS chk_exchange_rates_rate_type;

--bun:split
ALTER TABLE exchange_rates
    ADD COLUMN IF NOT EXISTS rate numeric(19,6);

--bun:split
UPDATE exchange_rates SET rate = selected_rate WHERE rate IS NULL;

--bun:split
ALTER TABLE exchange_rates
    ALTER COLUMN rate SET NOT NULL;

--bun:split
ALTER TABLE exchange_rates
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS rate_type,
    DROP COLUMN IF EXISTS bid,
    DROP COLUMN IF EXISTS ask,
    DROP COLUMN IF EXISTS mid,
    DROP COLUMN IF EXISTS selected_rate,
    DROP COLUMN IF EXISTS source_timestamp,
    DROP COLUMN IF EXISTS settlement_eligible;

--bun:split
ALTER TABLE exchange_rates
    ALTER COLUMN fetched_at TYPE bigint USING extract(epoch from fetched_at)::bigint,
    ALTER COLUMN fetched_at SET DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;

--bun:split
ALTER TABLE exchange_rates
    ADD CONSTRAINT uq_exchange_rates_org_currency_date UNIQUE ("organization_id", "business_unit_id", "from_currency", "to_currency", "date");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_exchange_rates_lookup" ON "exchange_rates"("organization_id", "business_unit_id", "from_currency", "to_currency", "date");

--bun:split
-- Note: ALTER TYPE ... REMOVE VALUE is not supported in PostgreSQL.
-- OANDAExchangeRates may remain in integration_type after rollback.
