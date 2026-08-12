-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260511000000_add_oanda_exchange_rates.tx.up.sql

CREATE TABLE IF NOT EXISTS "exchange_rates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL DEFAULT 'OANDA',
    "from_currency" TEXT NOT NULL,
    "to_currency" TEXT NOT NULL,
    "rate_type" TEXT NOT NULL DEFAULT 'mid',
    "bid" REAL NOT NULL,
    "ask" REAL NOT NULL,
    "mid" REAL NOT NULL,
    "selected_rate" REAL NOT NULL,
    "date" TEXT NOT NULL,
    "source_timestamp" TEXT NOT NULL,
    "fetched_at" TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "settlement_eligible" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "pk_exchange_rates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_exchange_rates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_exchange_rates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_exchange_rates_org_currency_date" UNIQUE ("organization_id", "business_unit_id", "provider", "from_currency", "to_currency", "rate_type", "date"),
    CONSTRAINT "chk_exchange_rates_currency_upper" CHECK (from_currency = upper(from_currency) AND to_currency = upper(to_currency)),
    CONSTRAINT "chk_exchange_rates_provider" CHECK (provider = 'OANDA'),
    CONSTRAINT "chk_exchange_rates_rate_type" CHECK (rate_type IN ('bid', 'ask', 'mid'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_exchange_rates_lookup" ON "exchange_rates" ("organization_id", "business_unit_id", "provider", "from_currency", "to_currency", "rate_type", "date");

--bun:split

CREATE TABLE IF NOT EXISTS "exchange_rate_settlement_quotes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL DEFAULT 'OANDA',
    "from_currency" TEXT NOT NULL,
    "to_currency" TEXT NOT NULL,
    "amount" REAL NOT NULL,
    "rate" REAL NOT NULL,
    "converted_amount" REAL NOT NULL,
    "rate_type" TEXT NOT NULL DEFAULT 'mid',
    "source_timestamp" TEXT NOT NULL,
    "fetched_at" TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expires_at" TEXT NOT NULL,
    CONSTRAINT "pk_exchange_rate_settlement_quotes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_exchange_rate_settlement_quotes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_exchange_rate_settlement_quotes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_exchange_rate_settlement_quotes_currency_upper" CHECK (from_currency = upper(from_currency) AND to_currency = upper(to_currency)),
    CONSTRAINT "chk_exchange_rate_settlement_quotes_provider" CHECK (provider = 'OANDA'),
    CONSTRAINT "chk_exchange_rate_settlement_quotes_rate_type" CHECK (rate_type IN ('bid', 'ask', 'mid'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_exchange_rate_settlement_quotes_lookup" ON "exchange_rate_settlement_quotes" ("organization_id", "business_unit_id", "from_currency", "to_currency", "fetched_at");
