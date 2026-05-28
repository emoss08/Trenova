ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'OANDAExchangeRates';

--bun:split
ALTER TYPE integration_category ADD VALUE IF NOT EXISTS 'FinancialData';

--bun:split
CREATE TABLE IF NOT EXISTS "exchange_rates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "provider" varchar(32) NOT NULL DEFAULT 'OANDA',
    "from_currency" varchar(3) NOT NULL,
    "to_currency" varchar(3) NOT NULL,
    "rate_type" varchar(16) NOT NULL DEFAULT 'mid',
    "bid" numeric(24,12) NOT NULL,
    "ask" numeric(24,12) NOT NULL,
    "mid" numeric(24,12) NOT NULL,
    "selected_rate" numeric(24,12) NOT NULL,
    "date" date NOT NULL,
    "source_timestamp" timestamptz NOT NULL,
    "fetched_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "settlement_eligible" boolean NOT NULL DEFAULT false,
    CONSTRAINT "pk_exchange_rates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_exchange_rates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_exchange_rates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_exchange_rates_org_currency_date" UNIQUE ("organization_id", "business_unit_id", "provider", "from_currency", "to_currency", "rate_type", "date"),
    CONSTRAINT "chk_exchange_rates_currency_upper" CHECK (from_currency = upper(from_currency) AND to_currency = upper(to_currency)),
    CONSTRAINT "chk_exchange_rates_provider" CHECK (provider = 'OANDA'),
    CONSTRAINT "chk_exchange_rates_rate_type" CHECK (rate_type IN ('bid', 'ask', 'mid'))
);

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
