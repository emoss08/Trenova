--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "ifta_fuel_type_enum" AS ENUM(
    'Diesel',
    'Gasoline',
    'Gasohol',
    'Propane',
    'CNG',
    'LNG',
    'Ethanol',
    'Methanol',
    'E85',
    'M85',
    'A55',
    'Biodiesel',
    'Electricity',
    'Hydrogen',
    'DEF',
    'Reefer',
    'Other'
);

--bun:split
CREATE TYPE "ifta_jurisdiction_status_enum" AS ENUM(
    'Active',
    'Inactive'
);

--bun:split
CREATE TABLE IF NOT EXISTS "ifta_jurisdictions"(
    "id" varchar(100) NOT NULL,
    "country_code" char(2) NOT NULL,
    "code" varchar(2) NOT NULL,
    "name" varchar(100) NOT NULL,
    "us_state_id" varchar(100),
    "is_ifta_member" boolean NOT NULL,
    "has_surcharge" boolean NOT NULL DEFAULT FALSE,
    "sort_order" integer NOT NULL DEFAULT 0,
    "status" ifta_jurisdiction_status_enum NOT NULL DEFAULT 'Active',
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ifta_jurisdictions" PRIMARY KEY ("id"),
    CONSTRAINT "uq_ifta_jurisdictions_code" UNIQUE ("country_code", "code"),
    CONSTRAINT "fk_ifta_jurisdictions_us_state" FOREIGN KEY ("us_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_ifta_jurisdictions_country_code" CHECK ("country_code" = upper("country_code") AND length("country_code") = 2),
    CONSTRAINT "chk_ifta_jurisdictions_code" CHECK ("code" = upper("code") AND length("code") = 2)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdictions_member" ON "ifta_jurisdictions"("is_ifta_member", "sort_order");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_jurisdictions_us_state" ON "ifta_jurisdictions"("us_state_id")
WHERE
    "us_state_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE ifta_jurisdictions IS 'Global reference table of the US states, DC and Canadian provinces an IFTA return can name. Member and non-member jurisdictions both exist so that every mile driven has a row to land on; only members carry tax rates.';

--bun:split
CREATE TABLE IF NOT EXISTS "ifta_tax_rates"(
    "id" varchar(100) NOT NULL,
    "jurisdiction_id" varchar(100) NOT NULL,
    "year" smallint NOT NULL,
    "quarter" smallint NOT NULL,
    "fuel_type" ifta_fuel_type_enum NOT NULL,
    "rate_per_gallon" numeric(10, 4) NOT NULL,
    "surcharge_rate_per_gallon" numeric(10, 4),
    "source_note" text,
    "source_url" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ifta_tax_rates" PRIMARY KEY ("id"),
    CONSTRAINT "uq_ifta_tax_rates_period" UNIQUE ("jurisdiction_id", "year", "quarter", "fuel_type"),
    CONSTRAINT "fk_ifta_tax_rates_jurisdiction" FOREIGN KEY ("jurisdiction_id") REFERENCES "ifta_jurisdictions"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_ifta_tax_rates_quarter" CHECK ("quarter" >= 1 AND "quarter" <= 4),
    CONSTRAINT "chk_ifta_tax_rates_rate" CHECK ("rate_per_gallon" >= 0),
    CONSTRAINT "chk_ifta_tax_rates_surcharge" CHECK ("surcharge_rate_per_gallon" IS NULL OR "surcharge_rate_per_gallon" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ifta_tax_rates_period" ON "ifta_tax_rates"("year", "quarter");

--bun:split
COMMENT ON TABLE ifta_tax_rates IS 'Global, admin-maintained IFTA tax rates per jurisdiction, quarter and fuel type, in USD per US gallon as published in the IFTA Inc. rate matrix. No rate is ever seeded: a missing rate is flagged on the return, never treated as zero.';
