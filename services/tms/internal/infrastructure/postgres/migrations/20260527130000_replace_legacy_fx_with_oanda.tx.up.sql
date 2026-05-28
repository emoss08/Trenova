ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'OANDAExchangeRates';

--bun:split
DELETE FROM integrations
WHERE category::text = 'FinancialData'
  AND type::text <> 'OANDAExchangeRates';

--bun:split
DELETE FROM exchange_rates;

--bun:split
ALTER TABLE exchange_rates
    ADD COLUMN IF NOT EXISTS provider varchar(32) NOT NULL DEFAULT 'OANDA',
    ADD COLUMN IF NOT EXISTS rate_type varchar(16) NOT NULL DEFAULT 'mid',
    ADD COLUMN IF NOT EXISTS bid numeric(24,12),
    ADD COLUMN IF NOT EXISTS ask numeric(24,12),
    ADD COLUMN IF NOT EXISTS mid numeric(24,12),
    ADD COLUMN IF NOT EXISTS selected_rate numeric(24,12),
    ADD COLUMN IF NOT EXISTS source_timestamp timestamptz,
    ADD COLUMN IF NOT EXISTS settlement_eligible boolean NOT NULL DEFAULT false;

--bun:split
ALTER TABLE exchange_rates
    ALTER COLUMN bid SET NOT NULL,
    ALTER COLUMN ask SET NOT NULL,
    ALTER COLUMN mid SET NOT NULL,
    ALTER COLUMN selected_rate SET NOT NULL,
    ALTER COLUMN source_timestamp SET NOT NULL;

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'exchange_rates'
          AND column_name = 'fetched_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE exchange_rates
            ALTER COLUMN fetched_at TYPE timestamptz USING to_timestamp(fetched_at);
    END IF;
END $$;

--bun:split
ALTER TABLE exchange_rates
    ALTER COLUMN fetched_at SET DEFAULT CURRENT_TIMESTAMP;

--bun:split
ALTER TABLE exchange_rates DROP COLUMN IF EXISTS rate;

--bun:split
ALTER TABLE exchange_rates DROP CONSTRAINT IF EXISTS uq_exchange_rates_org_currency_date;

--bun:split
ALTER TABLE exchange_rates
    ADD CONSTRAINT uq_exchange_rates_org_currency_date UNIQUE ("organization_id", "business_unit_id", "provider", "from_currency", "to_currency", "rate_type", "date");

--bun:split
ALTER TABLE exchange_rates
    DROP CONSTRAINT IF EXISTS chk_exchange_rates_provider,
    DROP CONSTRAINT IF EXISTS chk_exchange_rates_rate_type;

--bun:split
ALTER TABLE exchange_rates
    ADD CONSTRAINT chk_exchange_rates_provider CHECK (provider = 'OANDA'),
    ADD CONSTRAINT chk_exchange_rates_rate_type CHECK (rate_type IN ('bid', 'ask', 'mid'));

--bun:split
DROP INDEX IF EXISTS idx_exchange_rates_lookup;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_exchange_rates_lookup" ON "exchange_rates"("organization_id", "business_unit_id", "provider", "from_currency", "to_currency", "rate_type", "date");

--bun:split
CREATE TABLE IF NOT EXISTS "exchange_rate_settlement_quotes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" varchar(32) NOT NULL DEFAULT 'OANDA',
    "from_currency" varchar(3) NOT NULL,
    "to_currency" varchar(3) NOT NULL,
    "amount" numeric(24,8) NOT NULL,
    "rate" numeric(24,12) NOT NULL,
    "converted_amount" numeric(24,8) NOT NULL,
    "rate_type" varchar(16) NOT NULL DEFAULT 'mid',
    "source_timestamp" timestamptz NOT NULL,
    "fetched_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expires_at" timestamptz NOT NULL,
    CONSTRAINT "pk_exchange_rate_settlement_quotes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_exchange_rate_settlement_quotes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_exchange_rate_settlement_quotes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_exchange_rate_settlement_quotes_currency_upper" CHECK (from_currency = upper(from_currency) AND to_currency = upper(to_currency)),
    CONSTRAINT "chk_exchange_rate_settlement_quotes_provider" CHECK (provider = 'OANDA'),
    CONSTRAINT "chk_exchange_rate_settlement_quotes_rate_type" CHECK (rate_type IN ('bid', 'ask', 'mid'))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_exchange_rate_settlement_quotes_lookup" ON "exchange_rate_settlement_quotes"("organization_id", "business_unit_id", "from_currency", "to_currency", "fetched_at");
