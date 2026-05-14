ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'ExchangeRateAPI';

--bun:split
ALTER TYPE integration_category ADD VALUE IF NOT EXISTS 'FinancialData';

--bun:split
CREATE TABLE IF NOT EXISTS "exchange_rates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "from_currency" varchar(3) NOT NULL,
    "to_currency" varchar(3) NOT NULL,
    "rate" numeric(19,6) NOT NULL,
    "date" date NOT NULL,
    "fetched_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_exchange_rates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_exchange_rates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_exchange_rates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_exchange_rates_org_currency_date" UNIQUE ("organization_id", "business_unit_id", "from_currency", "to_currency", "date"),
    CONSTRAINT "chk_exchange_rates_currency_upper" CHECK (from_currency = upper(from_currency) AND to_currency = upper(to_currency))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_exchange_rates_lookup" ON "exchange_rates"("organization_id", "business_unit_id", "from_currency", "to_currency", "date");
